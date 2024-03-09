package request

import "transcode/pkg/resolution"

type TranscodeReq struct {
	InputUrl         string                  `json:"input_url"`
	FolderName       string                  `json:"folder_name"`
	FilePath         string                  `json:"file_path"`
	StoredFolderPath string                  `json:"stored_folder_path"`
	KeyInfoFilePath  string                  `json:"key_info_file_path"`
	Resolutions      []resolution.Resolution `json:"resolutions"`
}
