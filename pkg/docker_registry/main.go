package docker_registry

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"

	"github.com/flant/logboek"

	imagePkg "github.com/flant/werf/pkg/image"
)

var (
	InsecureRegistry      = false
	SkipTlsVerifyRegistry = false
	GCRUrlPatterns        = []string{"^container\\.cloud\\.google\\.com", "^gcr\\.io", "^.*\\.gcr\\.io"}
)

type RepoImage struct {
	Repository string
	Tag        string
	v1.Image
}

type Options struct {
	InsecureRegistry      bool
	SkipTlsVerifyRegistry bool
}

func Init(opts Options) error {
	InsecureRegistry = opts.InsecureRegistry
	SkipTlsVerifyRegistry = opts.SkipTlsVerifyRegistry

	if logboek.Debug.IsAccepted() {
		logs.Progress.SetOutput(logboek.GetOutStream())
		logs.Warn.SetOutput(logboek.GetErrStream())
		logs.Debug.SetOutput(logboek.GetOutStream())
	} else {
		logs.Progress.SetOutput(ioutil.Discard)
		logs.Warn.SetOutput(ioutil.Discard)
		logs.Debug.SetOutput(ioutil.Discard)
	}

	return nil
}

func IsGCR(reference string) (bool, error) {
	u, err := url.Parse(fmt.Sprintf("scheme://%s", reference))
	if err != nil {
		return false, err
	}

	for _, pattern := range GCRUrlPatterns {
		matched, err := regexp.MatchString(pattern, u.Host)
		if err != nil {
			return false, err
		}

		if matched {
			return true, nil
		}
	}

	return false, nil
}

func ImagesByWerfImageLabel(reference, labelValue string) ([]RepoImage, error) {
	var repoImages []RepoImage

	tags, err := Tags(reference)
	if err != nil {
		return nil, err
	}

	for _, tag := range tags {
		tagReference := strings.Join([]string{reference, tag}, ":")
		v1Image, _, err := image(tagReference)
		if err != nil {
			if strings.Contains(err.Error(), "MANIFEST_UNKNOWN") {
				logboek.LogWarnF("WARNING: Broken tag %s was skipped: %s\n", tagReference, err)
				continue
			}

			if strings.Contains(err.Error(), "BLOB_UNKNOWN") {
				logboek.LogWarnF("WARNING: Broken tag %s was skipped: %s\n", tagReference, err)
				continue
			}
			return nil, err
		}

		configFile, err := v1Image.ConfigFile()
		if err != nil {
			return nil, err
		}

		for k, v := range configFile.Config.Labels {
			if k == imagePkg.WerfImageLabel && v == labelValue {
				repoImage := RepoImage{
					Repository: reference,
					Tag:        tag,
					Image:      v1Image,
				}

				repoImages = append(repoImages, repoImage)
				break
			}
		}
	}

	return repoImages, nil
}

func Tags(reference string) ([]string, error) {
	tags, err := list(reference)
	if err != nil {
		if strings.Contains(err.Error(), "NAME_UNKNOWN") {
			return []string{}, nil
		}
		return nil, err
	}

	return tags, nil
}

func list(reference string) ([]string, error) {
	repo, err := name.NewRepository(reference, newRepositoryOptions()...)
	if err != nil {
		return nil, fmt.Errorf("parsing repo %q: %v", reference, err)
	}

	tags, err := remote.List(repo, remote.WithAuthFromKeychain(authn.DefaultKeychain), remote.WithTransport(getHttpTransport()))
	if err != nil {
		return nil, fmt.Errorf("reading tags for %q: %v", repo, err)
	}

	return tags, nil
}

func ImageId(reference string) (string, error) {
	i, _, err := image(reference)
	if err != nil {
		return "", err
	}

	manifest, err := i.Manifest()
	if err != nil {
		return "", err
	}

	return manifest.Config.Digest.String(), nil
}

func ImageParentId(reference string) (string, error) {
	configFile, err := ImageConfigFile(reference)
	if err != nil {
		return "", err
	}

	return configFile.Config.Image, nil
}

func ImageConfigFile(reference string) (v1.ConfigFile, error) {
	i, _, err := image(reference)
	if err != nil {
		return v1.ConfigFile{}, err
	}

	configFile, err := i.ConfigFile()
	if err != nil {
		return v1.ConfigFile{}, err
	}

	return *configFile, nil
}

func ImageDelete(reference string) error {
	r, err := name.ParseReference(reference, parseReferenceOptions()...)
	if err != nil {
		return fmt.Errorf("parsing reference %q: %v", reference, err)
	}

	if deleteErr := remote.Delete(r, remote.WithAuthFromKeychain(authn.DefaultKeychain), remote.WithTransport(getHttpTransport())); deleteErr != nil {
		if strings.Contains(deleteErr.Error(), "UNAUTHORIZED") {
			auth, authErr := authn.DefaultKeychain.Resolve(r.Context().Registry)
			if authErr != nil {
				return fmt.Errorf("getting creds for %q: %v", r, authErr)
			}

			if gitlabRegistryDeleteErr := GitlabRegistryDelete(r, auth, getHttpTransport()); gitlabRegistryDeleteErr != nil {
				if strings.Contains(gitlabRegistryDeleteErr.Error(), "UNAUTHORIZED") {
					return fmt.Errorf("deleting image %q: %v", r, deleteErr)
				}
				return fmt.Errorf("deleting image %q: %v", r, gitlabRegistryDeleteErr)
			}
		} else {
			return fmt.Errorf("deleting image %q: %v", r, deleteErr)
		}
	}

	return nil
}

// TODO https://gitlab.com/gitlab-org/gitlab-ce/issues/48968
func GitlabRegistryDelete(ref name.Reference, auth authn.Authenticator, t http.RoundTripper) error {
	scopes := []string{ref.Scope("*")}
	tr, err := transport.New(ref.Context().Registry, auth, t, scopes)
	if err != nil {
		return err
	}
	c := &http.Client{Transport: tr}

	u := url.URL{
		Scheme: ref.Context().Registry.Scheme(),
		Host:   ref.Context().RegistryStr(),
		Path:   fmt.Sprintf("/v2/%s/manifests/%s", ref.Context().RepositoryStr(), ref.Identifier()),
	}

	req, err := http.NewRequest(http.MethodDelete, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusAccepted:
		return nil
	default:
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("unrecognized status code during DELETE: %v; %v", resp.Status, string(b))
	}
}

func ImageDigest(reference string) (string, error) {
	i, _, err := image(reference)
	if err != nil {
		return "", err
	}

	digest, err := i.Digest()
	if err != nil {
		return "", err
	}

	return digest.String(), nil
}

func image(reference string) (v1.Image, name.Reference, error) {
	ref, err := name.ParseReference(reference, parseReferenceOptions()...)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing reference %q: %v", reference, err)
	}

	// FIXME: Hack for the go-containerregistry library,
	// FIXME: that uses default transport without options to change transport to custom.
	// FIXME: Needed for the insecure https registry to work.
	oldDefaultTransport := http.DefaultTransport
	http.DefaultTransport = getHttpTransport()
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	http.DefaultTransport = oldDefaultTransport

	if err != nil {
		return nil, nil, fmt.Errorf("reading image %q: %v", ref, err)
	}

	return img, ref, nil
}

func newRepositoryOptions() []name.Option {
	return parseReferenceOptions()
}

func parseReferenceOptions() []name.Option {
	var options []name.Option
	options = append(options, name.WeakValidation)

	if InsecureRegistry {
		options = append(options, name.Insecure)
	}

	return options
}

func getHttpTransport() (transport http.RoundTripper) {
	transport = http.DefaultTransport

	if SkipTlsVerifyRegistry {
		defaultTransport := http.DefaultTransport.(*http.Transport)

		newTransport := &http.Transport{
			Proxy:                 defaultTransport.Proxy,
			DialContext:           defaultTransport.DialContext,
			MaxIdleConns:          defaultTransport.MaxIdleConns,
			IdleConnTimeout:       defaultTransport.IdleConnTimeout,
			TLSHandshakeTimeout:   defaultTransport.TLSHandshakeTimeout,
			ExpectContinueTimeout: defaultTransport.ExpectContinueTimeout,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
			TLSNextProto:          make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
		}

		transport = newTransport
	}

	return
}
