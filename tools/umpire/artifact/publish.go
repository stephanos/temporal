package artifact

import (
	"errors"
	"fmt"
	"strings"

	"go.temporal.io/server/tools/common/artifactio"
)

const artifactSetManifestPath = "manifest.json"

var artifactSetDirectory = artifactio.ImmutableDirectory{
	ManifestPath:     artifactSetManifestPath,
	MaximumFileBytes: MaximumDocumentBytes,
	MemberPaths:      artifactSetMemberPaths,
	Validate:         validatePublishedArtifactSet,
}

// PublishSet publishes an admitted set at root/sets/<manifest-sha256> and
// returns that immutable digest directory.
func PublishSet(root string, admitted AdmittedSet) (string, error) {
	files, err := admittedSetFiles(admitted)
	if err != nil {
		return "", err
	}
	digest := strings.TrimPrefix(admitted.ManifestSHA256(), "sha256:")
	return artifactSetDirectory.Publish(root, digest, files)
}

// LoadSet admits one exact immutable digest directory after rehashing its
// manifest and every member from regular files opened without symlink traversal.
func LoadSet(destination string) (AdmittedSet, error) {
	files, err := artifactSetDirectory.Read(destination)
	if err != nil {
		return AdmittedSet{}, err
	}
	return admitPublishedArtifactSet(files)
}

func admittedSetFiles(admitted AdmittedSet) (map[string][]byte, error) {
	if len(admitted.members) == 0 || len(admitted.manifestBytes) == 0 || admitted.manifestSHA256 == "" {
		return nil, errors.New("admitted Artifact set is required")
	}
	files := make(map[string][]byte, len(admitted.members)+1)
	files[artifactSetManifestPath] = admitted.ManifestBytes()
	for _, member := range admitted.members {
		files[member.Path] = append([]byte(nil), member.Encoded...)
	}
	if err := validatePublishedArtifactSet(files); err != nil {
		return nil, fmt.Errorf("validate admitted Artifact set: %w", err)
	}
	return files, nil
}

func artifactSetMemberPaths(encodedManifest []byte) ([]string, error) {
	manifest, err := artifactSetManifestDecoder.Decode(encodedManifest)
	if err != nil {
		return nil, err
	}
	paths := make([]string, len(manifest.Members))
	for index, member := range manifest.Members {
		paths[index] = member.Path
	}
	return paths, nil
}

func validatePublishedArtifactSet(files map[string][]byte) error {
	_, err := admitPublishedArtifactSet(files)
	return err
}

func admitPublishedArtifactSet(files map[string][]byte) (AdmittedSet, error) {
	encodedManifest, exists := files[artifactSetManifestPath]
	if !exists {
		return AdmittedSet{}, errors.New("published Artifact set has no manifest")
	}
	manifest, err := artifactSetManifestDecoder.Decode(encodedManifest)
	if err != nil {
		return AdmittedSet{}, err
	}
	members := make([]SetMember, len(manifest.Members))
	for index, row := range manifest.Members {
		encoded, exists := files[row.Path]
		if !exists {
			return AdmittedSet{}, wrapAdmission(ErrorClosure,
				fmt.Errorf("published Artifact set is missing member %q", row.Path))
		}
		members[index] = SetMember{Path: row.Path, Encoded: append([]byte(nil), encoded...)}
	}
	if len(files) != len(members)+1 {
		return AdmittedSet{}, wrapAdmission(ErrorClosure,
			errors.New("published Artifact set contains unexpected files"))
	}
	return AdmitSetManifest(encodedManifest, members)
}
