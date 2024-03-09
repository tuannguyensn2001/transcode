package config

type Base struct {
	HTTPAddress       string `json:"http_address" mapstructure:"http_address"  validate:"required"`
	Environment       string `json:"environment" mapstructure:"environment"  validate:"required"`
	ServiceName       string `json:"service_name" mapstructure:"service_name"`
	LogLevel          string `json:"log_level" mapstructure:"log_level"`
	LogColor          bool   `json:"log_color" mapstructure:"log_color"`
	LogFilePath       string `json:"log_file_path" mapstructure:"log_file_path"`
	LogFileSize       int    `json:"log_file_size" mapstructure:"log_file_size"` // in MB
	LogFileAge        int    `json:"log_file_age" mapstructure:"log_file_age"`   // in days
	LogFileBackups    int    `json:"log_file_backups" mapstructure:"log_file_backups"`
	LogTelegramEnable bool   `json:"log_telegram_enable" mapstructure:"log_telegram_enable"`
	LogTelegramToken  string `json:"log_telegram_token" mapstructure:"log_telegram_token"`
	LogTelegramChatID int64  `json:"log_telegram_chat_id" mapstructure:"log_telegram_chat_id"`
}

type Config struct {
	Base         `mapstructure:",squash"`
	SentryConfig SentryConfig `json:"sentry" mapstructure:"sentry"`
	MaxPoolSize  int          `json:"max_pool_size" mapstructure:"max_pool_size"`

	ServerConfig ServerConfig `json:"server" mapstructure:"server"`
	KafkaConfig  KafkaConfig  `json:"kafka" mapstructure:"kafka"`
}

// SentryConfig ...
type SentryConfig struct {
	Enabled bool   `json:"enabled" mapstructure:"enabled"`
	DNS     string `json:"dns" mapstructure:"dns"`
	Trace   bool   `json:"trace" mapstructure:"trace"`
}

type KafkaConfig struct {
	Username string `json:"username" mapstructure:"username"`
	Password string `json:"password" mapstructure:"password"`
	Brokers  string `json:"brokers" mapstructure:"brokers"`
	//CommitType      broker.CommitType `json:"commit_type" mapstructure:"commit_type"`
	AutoCreateTopic bool `json:"auto_create_topic" mapstructure:"auto_create_topic"`

	CancelledTopicGroupID string `json:"cancelled_topic_group_id" mapstructure:"cancelled_topic_group_id"`

	TranscodeJobCreatedTopic   string `json:"transcode_job_created_topic" mapstructure:"transcode_job_created_topic"`
	TranscodeJobFinishedTopic  string `json:"transcode_job_finished_topic" mapstructure:"transcode_job_finished_topic"`
	TranscodeJobCancelledTopic string `json:"transcode_job_cancelled_topic" mapstructure:"transcode_job_cancelled_topic"`
}

type ServerConfig struct {
	FfmpegBin                        string `json:"ffmpeg_bin" mapstructure:"ffmpeg_bin"`
	FfprobeBin                       string `json:"ffprobe_bin" mapstructure:"ffprobe_bin"`
	OutputPath                       string `json:"output_path" mapstructure:"output_path"`
	DownloadPath                     string `json:"download_path" mapstructure:"download_path"`
	ClearAfterStream                 bool   `json:"clear_after_stream" mapstructure:"clear_after_stream"`
	TranscoderVersion                int    `json:"transcoder_version" mapstructure:"transcoder_version"`
	GoogleCloudBucket                string `json:"google_cloud_bucket" mapstructure:"google_cloud_bucket"`
	GoogleCloudStorageCredentialPath string `json:"google_cloud_storage_credential_path" mapstructure:"google_cloud_storage_credential_path"`
	GoogleDriveCredentialPath        string `json:"google_drive_credential_path" mapstructure:"google_drive_credential_path"`
	Default1080Bitrate               int64  `json:"default_1080_bitrate" mapstructure:"default_1080_bitrate"`
	IgnoreBitrateThreshold           int64  `json:"ignore_bitrate_threshold" mapstructure:"ignore_bitrate_threshold"`
	TargetSegmentDuration            int    `json:"target_segment_duration" mapstructure:"target_segment_duration"`
}
