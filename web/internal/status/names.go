package status

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	peerDirPrefix     = "peer"
	peerDisplayPrefix = "peer_"
	shortKeyLen       = 8
)

type peerFile struct {
	Name   string
	Config string
}

func displayPeerName(dir string) string {
	if strings.HasPrefix(dir, peerDisplayPrefix) {
		if name := strings.TrimPrefix(dir, peerDisplayPrefix); name != "" {
			return name
		}
	}
	return dir
}

func shortPublicKey(pub string) string {
	if len(pub) <= shortKeyLen {
		return pub
	}
	return pub[:shortKeyLen]
}

func loadPeerFiles(configDir string) (map[string]peerFile, error) {
	entries, err := os.ReadDir(configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]peerFile{}, nil
		}
		return nil, err
	}

	out := make(map[string]peerFile)
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), peerDirPrefix) {
			continue
		}
		dir := entry.Name()
		pub, err := os.ReadFile(filepath.Join(configDir, dir, "publickey-"+dir))
		if err != nil {
			continue
		}
		key := strings.TrimSpace(string(pub))
		if key == "" {
			continue
		}
		configText, err := os.ReadFile(filepath.Join(configDir, dir, dir+".conf"))
		if err != nil {
			configText = nil
		}
		out[key] = peerFile{
			Name:   displayPeerName(dir),
			Config: string(configText),
		}
	}
	return out, nil
}
