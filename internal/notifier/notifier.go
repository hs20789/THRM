package notifier

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/TIANLI0/THRM/internal/appmeta"
	"github.com/TIANLI0/THRM/internal/types"
	"github.com/TIANLI0/THRM/internal/uilocale"
	"github.com/gen2brain/beeep"
)

// Manager 使用 beeep 发送系统通知。
type Manager struct {
	logger   types.Logger
	iconPath string
	send     func(string, string, any) error
}

func NewManager(logger types.Logger, iconData []byte) *Manager {
	beeep.AppName = appmeta.AppName
	return &Manager{
		logger:   logger,
		iconPath: ensureNotificationIcon(iconData, logger),
		send:     beeep.Notify,
	}
}

func (m *Manager) Notify(locale uilocale.Snapshot, title, message string) {
	title = strings.TrimSpace(title)
	if strings.TrimSpace(message) == "" {
		return
	}

	toastTitle := locale.Text("nativeUI.notification.fallback", nil)
	if title != "" {
		toastTitle = title
	}

	if err := m.send(toastTitle, message, m.iconPath); err != nil {
		if m.logger != nil {
			m.logger.Debug("系统通知发送失败: %v", err)
		}
	}
}

func ensureNotificationIcon(iconData []byte, logger types.Logger) string {
	if len(iconData) == 0 {
		return ""
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil || cacheDir == "" {
		cacheDir = os.TempDir()
	}

	iconDir := filepath.Join(cacheDir, appmeta.NotificationCacheDir)
	if err := os.MkdirAll(iconDir, 0755); err != nil {
		if logger != nil {
			logger.Debug("创建通知图标缓存目录失败: %v", err)
		}
		return ""
	}

	iconPath := filepath.Join(iconDir, "notify-icon.ico")
	if err := os.WriteFile(iconPath, iconData, 0644); err != nil {
		if logger != nil {
			logger.Debug("写入通知图标缓存失败: %v", err)
		}
		return ""
	}

	return iconPath
}
