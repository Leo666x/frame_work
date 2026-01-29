package minio_mw

import (
	"context"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"net/url"
	"time"
)

type Minio struct {
	client *minio.Client
	config *Config
}
type Config struct {
	Ip   string
	Port string
	Ak   string
	Sk   string
}

func New(c *Config) (*Minio, error) {
	minioEndPoint := fmt.Sprintf("%s:%s", c.Ip, c.Port)
	client, err := minio.New(minioEndPoint, &minio.Options{
		Creds:  credentials.NewStaticV4(c.Ak, c.Sk, ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}

	return &Minio{client: client, config: c}, nil
}

func (m *Minio) check() error {
	if m.client == nil {
		return fmt.Errorf("minio初始化失败,endpoints：%s:%s", m.config.Ip, m.config.Port)
	}
	return nil
}

func (m *Minio) UpLoad(bucketName, bucketFilePath, uploadFile string) error {
	location := "us-east-1"

	//1.判断桶存不存在
	exists, err := m.client.BucketExists(context.Background(), bucketName)
	if err != nil {
		//写日志
		return err
	}
	//2.不存在创建
	if !exists {
		err = m.client.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{Region: location})
		if err != nil {
			return err
		}
	}
	//3.上传文件
	contentType := "application/zip"
	_, err = m.client.FPutObject(context.Background(), bucketName, bucketFilePath, uploadFile, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		//上传失败，写日志
		return err
	}
	return nil
}

func (m *Minio) UpLoadF(bucketName, bucketFilePath string, src io.Reader, size int64) (minio.UploadInfo, error) {
	location := "us-east-1"

	//1.判断桶存不存在
	exists, err := m.client.BucketExists(context.Background(), bucketName)
	if err != nil {
		//写日志
		return minio.UploadInfo{}, err
	}
	//2.不存在创建
	if !exists {
		err = m.client.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{Region: location})
		if err != nil {
			return minio.UploadInfo{}, err
		}
	}
	//3.上传文件
	contentType := "application/octet-stream"
	info, err := m.client.PutObject(context.Background(), bucketName, bucketFilePath, src, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		//上传失败，写日志
		return minio.UploadInfo{}, err
	}
	return info, nil
}

func (m *Minio) DownLoad(bucketName, bucketFilePath, savePath string) error {
	return m.client.FGetObject(context.Background(), bucketName, bucketFilePath, savePath, minio.GetObjectOptions{})
}

func (m *Minio) GetFileURL(bucketName, bucketFilePath string, expires time.Duration) (*url.URL, error) {
	return m.client.PresignedGetObject(context.Background(), bucketName, bucketFilePath, 1*time.Hour, url.Values{})
}
