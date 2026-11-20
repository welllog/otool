package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	DriverLocal  = "local"
	DriverWebDAV = "webdav"
	DriverS3     = "s3"

	configSubDir   = ".otool"
	configFileName = "storage_profiles.json"
)

type LocalConfig struct {
	DefaultPath string `json:"default_path"`
}

type WebDAVConfig struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"` // Encrypted or plain
	RootPath string `json:"root_path"`
}

type S3Config struct {
	Endpoint  string `json:"endpoint"`
	Region    string `json:"region"`
	Bucket    string `json:"bucket"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"` // Encrypted or plain
	Prefix    string `json:"prefix"`
}

type Profile struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Type      string       `json:"type"` // "local", "webdav", "s3"
	Local     LocalConfig  `json:"local,omitempty"`
	WebDAV    WebDAVConfig `json:"webdav,omitempty"`
	S3        S3Config     `json:"s3,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

type ProfilesFile struct {
	Version    int       `json:"version"`
	LastUsedID string    `json:"last_used_id"`
	Profiles   []Profile `json:"profiles"`
}

var (
	profileMu  sync.RWMutex
	cachedKey  []byte
	keyMu      sync.Mutex
	keyLoaded  bool
	keyLoadErr error
)

// GetConfigPath returns the absolute path to the configuration file (~/.otool/storage_profiles.json).
func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, configSubDir, configFileName)
}

// GetConfigDir returns the directory containing the configuration file.
func GetConfigDir() string {
	return filepath.Dir(GetConfigPath())
}

// OpenConfigFolder opens the directory containing the config file in the OS file manager.
func OpenConfigFolder() error {
	dir := GetConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	_ = os.Chmod(dir, 0700)

	cfgPath := GetConfigPath()
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		if _, err := os.Stat(cfgPath); err == nil {
			cmd = exec.Command("open", "-R", cfgPath)
		} else {
			cmd = exec.Command("open", dir)
		}
	case "windows":
		if _, err := os.Stat(cfgPath); err == nil {
			cmd = exec.Command("explorer.exe", "/select,"+cfgPath)
		} else {
			cmd = exec.Command("explorer.exe", dir)
		}
	default: // linux, freebsd, etc.
		cmd = exec.Command("xdg-open", dir)
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// getMachineSecretKey returns the AES-256 key used for local secrets. A key
// file next to the profile config keeps the key stable if the machine hostname
// or home directory changes. New configurations always get a random key; the
// legacy hostname/home derivation is used only to migrate an existing config
// that already contains encrypted secrets but has no key file yet.
func configHasEncryptedSecrets() bool {
	data, err := os.ReadFile(GetConfigPath())
	if err != nil {
		return false
	}
	var pf ProfilesFile
	if err := json.Unmarshal(data, &pf); err != nil {
		return false
	}
	for _, p := range pf.Profiles {
		if strings.HasPrefix(p.WebDAV.Password, "enc:") || strings.HasPrefix(p.S3.SecretKey, "enc:") {
			return true
		}
	}
	return false
}

func writeSecretKey(key []byte) error {
	dir := GetConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	_ = os.Chmod(dir, 0700)
	keyPath := filepath.Join(dir, "secret.key")
	tmpPath := keyPath + ".tmp"
	if err := os.WriteFile(tmpPath, key, 0600); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0600); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, keyPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	_ = os.Chmod(keyPath, 0600)
	return nil
}

func getMachineSecretKey() ([]byte, error) {
	keyMu.Lock()
	defer keyMu.Unlock()
	if keyLoaded {
		return cachedKey, keyLoadErr
	}
	keyLoaded = true

	keyPath := filepath.Join(GetConfigDir(), "secret.key")
	if data, err := os.ReadFile(keyPath); err == nil {
		if len(data) < 32 {
			keyLoadErr = fmt.Errorf("secret.key 文件长度不足 32 字节，可能已损坏")
			return nil, keyLoadErr
		}
		if err := os.Chmod(keyPath, 0600); err != nil {
			keyLoadErr = fmt.Errorf("收紧 secret.key 权限失败: %w", err)
			return nil, keyLoadErr
		}
		cachedKey = append([]byte(nil), data[:32]...)
		return cachedKey, nil
	} else if !os.IsNotExist(err) {
		keyLoadErr = fmt.Errorf("读取 secret.key 失败: %w", err)
		return nil, keyLoadErr
	}

	if !configHasEncryptedSecrets() {
		key := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			keyLoadErr = fmt.Errorf("生成 secret.key 失败: %w", err)
			return nil, keyLoadErr
		}
		if err := writeSecretKey(key); err != nil {
			keyLoadErr = fmt.Errorf("写入 secret.key 失败: %w", err)
			return nil, keyLoadErr
		}
		cachedKey = key
		return cachedKey, nil
	}

	hostname, _ := os.Hostname()
	user, _ := os.UserHomeDir()
	seed := fmt.Sprintf("otool-salt-v1:%s:%s", hostname, user)
	legacy := sha256.Sum256([]byte(seed))
	cachedKey = append([]byte(nil), legacy[:]...)
	if err := writeSecretKey(cachedKey); err != nil {
		keyLoadErr = fmt.Errorf("写入迁移密钥 secret.key 失败: %w", err)
		return nil, keyLoadErr
	}
	return cachedKey, nil
}

// EncryptSecret encrypts sensitive fields like passwords with AES-GCM and base64.
func EncryptSecret(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	key, err := getMachineSecretKey()
	if err != nil {
		return "", fmt.Errorf("初始化密码加密失败: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("初始化密码加密失败: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("初始化密码加密失败: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("生成密码加密随机数失败: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return "enc:" + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptSecret decrypts an encrypted secret string. If not prefixed with "enc:", returns original.
func DecryptSecret(encoded string) (string, error) {
	if !strings.HasPrefix(encoded, "enc:") {
		return encoded, nil
	}
	raw := strings.TrimPrefix(encoded, "enc:")
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", fmt.Errorf("密码编码格式无效: %w", err)
	}
	key, err := getMachineSecretKey()
	if err != nil {
		return "", fmt.Errorf("密码解密失败: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("密码解密失败: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("密码解密失败: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("密码密文长度无效")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("密码解密失败（配置可能来自其他机器）: %w", err)
	}
	return string(plaintext), nil
}

// LoadProfiles reads profiles from disk. If file doesn't exist, returns default initial profiles.
func LoadProfiles() (*ProfilesFile, error) {
	profileMu.RLock()
	defer profileMu.RUnlock()

	cfgPath := GetConfigPath()
	data, err := os.ReadFile(cfgPath)
	if err == nil {
		_ = os.Chmod(cfgPath, 0600)
	}
	if err != nil {
		if os.IsNotExist(err) {
			// Initial default template with Local profile
			return &ProfilesFile{
				Version:    1,
				LastUsedID: "local_default",
				Profiles: []Profile{
					{
						ID:        "local_default",
						Name:      "本地磁盘",
						Type:      DriverLocal,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					},
				},
			}, nil
		}
		return nil, fmt.Errorf("读取配置文件失败 (%s): %w", cfgPath, err)
	}

	var pf ProfilesFile
	if err := json.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("配置文件格式错误 (%s): %w", cfgPath, err)
	}
	return &pf, nil
}

// SaveProfiles writes the profiles to disk with 0600 permissions.
func SaveProfiles(pf *ProfilesFile) error {
	profileMu.Lock()
	defer profileMu.Unlock()

	dir := GetConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	_ = os.Chmod(dir, 0700)

	data, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		return err
	}

	cfgPath := GetConfigPath()
	tmpPath := cfgPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, cfgPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	_ = os.Chmod(cfgPath, 0600)
	return nil
}
