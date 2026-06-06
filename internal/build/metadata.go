package build

import (
	"encoding/json"
	"os"
	"strings"
)

// Metadata builds a buildx-compatible metadata map from the image ID read from
// the iidfile and the primary image reference. buildah's iidfile contains the
// image ID (a config digest); we surface it under the keys tools commonly read.
func Metadata(iidPath, primaryRef string) map[string]any {
	meta := map[string]any{}
	if iidPath != "" {
		if b, err := os.ReadFile(iidPath); err == nil {
			id := strings.TrimSpace(string(b))
			id = strings.TrimPrefix(id, "sha256:")
			if id != "" {
				meta["containerimage.digest"] = "sha256:" + id
				meta["containerimage.config.digest"] = "sha256:" + id
			}
		}
	}
	if primaryRef != "" {
		meta["image.name"] = primaryRef
	}
	return meta
}

// WriteMetadataFile writes a single build's metadata to path.
func WriteMetadataFile(path, iidPath, primaryRef string) error {
	return writeJSON(path, Metadata(iidPath, primaryRef))
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

// WriteBakeMetadataFile writes a per-target metadata map (target name → metadata)
// to path, matching the shape of `buildx bake --metadata-file`.
func WriteBakeMetadataFile(path string, byTarget map[string]map[string]any) error {
	return writeJSON(path, byTarget)
}
