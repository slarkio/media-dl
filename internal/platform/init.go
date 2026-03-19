package platform

// PlatformAdapter must be implemented by all platform adapters.
// All adapters must implement BuildDownloadCommand and BuildMetadataURL.
// XiaoyuzhouAdapter also implements AudioURLResolver for audio URL resolution.

func init() {
	RegisterAdapter(NewYouTubeAdapter())
	RegisterAdapter(NewBilibiliAdapter())
	RegisterAdapter(NewXiaoyuzhouAdapter())
}
