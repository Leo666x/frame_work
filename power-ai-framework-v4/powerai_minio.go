package powerai

import (
	"github.com/minio/minio-go/v7"
	"io"
	"net/url"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/minio"
	"time"
)

// ***************************************************************************************************************
//  minio相关方法
// ***************************************************************************************************************

func (a *AgentApp) GetMinioClient() (*minio_mw.Minio, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.minio == nil {
		client, err := initMinio(a.etcd)
		if err != nil {
			return nil, err
		}
		a.minio = client
	}
	return a.minio, nil
}

// UpLoadToMinio 上传到minio
// 参数：
//
//	bucketName 桶名称
//	bucketFilePath minio上存储的路径
//	uploadFile  待上传的文件，不可是文件夹
func (a *AgentApp) UpLoadToMinio(enterpriseId, bucketName, bucketFilePath, uploadFile string) error {
	client, err := a.GetMinioClient()
	if err != nil {
		return err
	}
	return client.UpLoad(bucketName, bucketFilePath, uploadFile)
}

// UpLoadToMinioF 上传到minio(指定文件，io转发)
// 参数：
//
//	bucketName 桶名称
//	bucketFilePath minio上存储的路径
//	uploadFile  待上传的文件，不可是文件夹
func (a *AgentApp) UpLoadToMinioF(enterpriseId, bucketName, bucketFilePath string, src io.Reader, size int64) (minio.UploadInfo, error) {
	client, err := a.GetMinioClient()
	if err != nil {
		return minio.UploadInfo{}, err
	}
	return client.UpLoadF(bucketName, bucketFilePath, src, size)

}

// DownLoadFromMinio 从minio下载文件
// 参数：
//
//	bucketName 桶名称
//	bucketFilePath minio上存储的路径
//	savePath  本地保存的路径，不可以是文件夹
func (a *AgentApp) DownLoadFromMinio(enterpriseId, bucketName, bucketFilePath, savePath string) error {
	client, err := a.GetMinioClient()
	if err != nil {
		return err
	}
	return client.DownLoad(bucketName, bucketFilePath, savePath)
}

func (a *AgentApp) GetFileURL(enterpriseId, bucketName, bucketFilePath string, savePath time.Duration) (*url.URL, error) {
	client, err := a.GetMinioClient()
	if err != nil {
		return nil, err
	}
	return client.GetFileURL(bucketName, bucketFilePath, savePath)
}
