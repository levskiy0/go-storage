package go_storage

import (
	"context"
	"fmt"
	filesystem "github.com/levskiy0/go-storage/contract"
	"github.com/levskiy0/go-storage/file"
	"github.com/minio/minio-go/v7"
	"io"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/golang-module/carbon/v2"
	"github.com/gookit/color"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

/*
 * MinIO OSS
 * Document: https://min.io/docs/minio/linux/developers/go/minio-go.html
 * Example: https://github.com/minio/minio-go/tree/master/examples/s3
 */

type MinioConfig struct {
	Key      string
	Secret   string
	Region   string
	Bucket   string
	Ssl      bool
	Url      string
	Endpoint string
}

type Minio struct {
	ctx      context.Context
	config   MinioConfig
	instance *minio.Client
	bucket   string
	url      string
}

func NewMinio(ctx context.Context, config MinioConfig) (*Minio, error) {
	key := config.Key
	secret := config.Secret
	region := config.Region
	bucket := config.Bucket
	diskUrl := config.Url
	ssl := config.Ssl
	endpoint := config.Endpoint

	if key == "" || secret == "" || bucket == "" || diskUrl == "" || endpoint == "" {
		return nil, fmt.Errorf("please set configuration first")
	}

	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimPrefix(endpoint, "https://")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(key, secret, ""),
		Secure: ssl,
		Region: region,
	})
	if err != nil {
		return nil, fmt.Errorf("init driver error: %s", err)
	}

	return &Minio{
		ctx:      ctx,
		config:   config,
		instance: client,
		bucket:   bucket,
		url:      diskUrl,
	}, nil
}

func (r *Minio) AllDirectories(path string) ([]string, error) {
	var directories []string
	validPath := validPathMinio(path)
	objectCh := r.instance.ListObjects(r.ctx, r.bucket, minio.ListObjectsOptions{
		Prefix:    validPath,
		Recursive: false,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}

		if strings.HasSuffix(object.Key, "/") {
			key := strings.TrimPrefix(object.Key, validPath)
			if key != "" {
				directories = append(directories, key)
				subDirectories, err := r.AllDirectories(object.Key)
				if err != nil {
					return nil, err
				}
				for _, subDirectory := range subDirectories {
					directories = append(directories, strings.TrimPrefix(object.Key+subDirectory, validPath))
				}
			}
		}
	}

	return directories, nil
}

func (r *Minio) AllFiles(path string) ([]string, error) {
	var files []string
	validPath := validPathMinio(path)

	objectCh := r.instance.ListObjects(r.ctx, r.bucket, minio.ListObjectsOptions{
		Prefix:    validPath,
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}

		if !strings.HasSuffix(object.Key, "/") {
			files = append(files, strings.TrimPrefix(object.Key, validPath))
		}
	}

	return files, nil
}

func (r *Minio) Copy(originFile, targetFile string) error {
	srcOpts := minio.CopySrcOptions{
		Bucket: r.bucket,
		Object: originFile,
	}
	dstOpts := minio.CopyDestOptions{
		Bucket: r.bucket,
		Object: targetFile,
	}
	_, err := r.instance.CopyObject(r.ctx, dstOpts, srcOpts)
	return err
}

func (r *Minio) Delete(files ...string) error {
	objectsCh := make(chan minio.ObjectInfo, len(files))
	go func() {
		defer close(objectsCh)
		for _, file := range files {
			object := minio.ObjectInfo{
				Key: file,
			}
			objectsCh <- object
		}
	}()

	for err := range r.instance.RemoveObjects(r.ctx, r.bucket, objectsCh, minio.RemoveObjectsOptions{}) {
		return err.Err
	}

	return nil
}

func (r *Minio) DeleteDirectory(directory string) error {
	if !strings.HasSuffix(directory, "/") {
		directory += "/"
	}
	opts := minio.RemoveObjectOptions{
		ForceDelete: true,
	}
	err := r.instance.RemoveObject(r.ctx, r.bucket, directory, opts)
	if err != nil {
		return err
	}

	return nil
}

func (r *Minio) Directories(path string) ([]string, error) {
	var directories []string
	validPath := validPathMinio(path)
	objectCh := r.instance.ListObjects(r.ctx, r.bucket, minio.ListObjectsOptions{
		Prefix:    validPath,
		Recursive: false,
	})
	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}
		if strings.HasSuffix(object.Key, "/") {
			directories = append(directories, strings.ReplaceAll(object.Key, validPath, ""))
		}
	}

	return directories, nil
}

func (r *Minio) Exists(file string) bool {
	_, err := r.instance.StatObject(r.ctx, r.bucket, file, minio.StatObjectOptions{})

	return err == nil
}

func (r *Minio) Files(path string) ([]string, error) {
	var files []string
	validPath := validPathMinio(path)

	for object := range r.instance.ListObjects(r.ctx, r.bucket, minio.ListObjectsOptions{
		Prefix:    validPath,
		Recursive: false,
	}) {
		if object.Err != nil {
			return nil, object.Err
		}
		if !strings.HasSuffix(object.Key, "/") {
			files = append(files, strings.ReplaceAll(object.Key, validPath, ""))
		}
	}

	return files, nil
}

func (r *Minio) Get(file string) (string, error) {
	data, err := r.GetBytes(file)

	return string(data), err
}

func (r *Minio) GetBytes(file string) ([]byte, error) {
	object, err := r.instance.GetObject(r.ctx, r.bucket, file, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(object)
	defer object.Close()
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (r *Minio) LastModified(file string) (time.Time, error) {
	objInfo, err := r.instance.StatObject(r.ctx, r.bucket, file, minio.StatObjectOptions{})
	if err != nil {
		return time.Time{}, err
	}

	l, err := time.LoadLocation(storage.timezone)
	if err != nil {
		return time.Time{}, err
	}

	return objInfo.LastModified.In(l), nil
}

func (r *Minio) MakeDirectory(directory string) error {
	if !strings.HasSuffix(directory, "/") {
		directory += "/"
	}

	return r.Put(directory, "")
}

func (r *Minio) MimeType(file string) (string, error) {
	objInfo, err := r.instance.StatObject(r.ctx, r.bucket, file, minio.StatObjectOptions{})
	if err != nil {
		return "", err
	}

	return objInfo.ContentType, nil
}

func (r *Minio) Missing(file string) bool {
	return !r.Exists(file)
}

func (r *Minio) Move(oldFile, newFile string) error {
	if err := r.Copy(oldFile, newFile); err != nil {
		return err
	}

	return r.Delete(oldFile)
}

func (r *Minio) Path(file string) string {
	return file
}

func (r *Minio) Put(file string, content string) error {
	mtype := mimetype.Detect([]byte(content))
	reader := strings.NewReader(content)
	_, err := r.instance.PutObject(
		r.ctx,
		r.bucket,
		file,
		reader,
		reader.Size(),
		minio.PutObjectOptions{
			ContentType: mtype.String(),
		},
	)

	return err
}

func (r *Minio) PutFile(filePath string, source filesystem.File) (string, error) {
	return r.PutFileAs(filePath, source, Random(40))
}

func (r *Minio) PutFileAs(filePath string, source filesystem.File, name string) (string, error) {
	fullPath, err := fullPathOfFileMinio(filePath, source, name)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(source.File())
	if err != nil {
		return "", err
	}

	if err := r.Put(fullPath, string(data)); err != nil {
		return "", err
	}

	return fullPath, nil
}

func (r *Minio) Size(file string) (int64, error) {
	objInfo, err := r.instance.StatObject(r.ctx, r.bucket, file, minio.StatObjectOptions{})
	if err != nil {
		return 0, err
	}

	return objInfo.Size, nil
}

func (r *Minio) TemporaryUrl(file string, time time.Time) (string, error) {
	file = strings.TrimPrefix(file, "/")
	reqParams := make(url.Values)
	presignedURL, err := r.instance.PresignedGetObject(r.ctx, r.bucket, file, time.Sub(carbon.Now().StdTime()), reqParams)
	if err != nil {
		return "", err
	}

	return presignedURL.String(), nil
}

func (r *Minio) WithContext(ctx context.Context) filesystem.Driver {
	driver, err := NewMinio(ctx, r.config)
	if err != nil {
		color.Redf("init driver fail: %v\n", err)

		return nil
	}

	return driver
}

func (r *Minio) Url(file string) string {
	realUrl := strings.TrimSuffix(r.url, "/")
	if !strings.HasSuffix(realUrl, r.bucket) {
		realUrl += "/" + r.bucket
	}

	return realUrl + "/" + strings.TrimPrefix(file, "/")
}

func fullPathOfFileMinio(filePath string, source filesystem.File, name string) (string, error) {
	extension := path.Ext(name)
	if extension == "" {
		var err error
		extension, err = file.Extension(source.File(), true)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("%s/%s.%s", filePath, strings.TrimSuffix(strings.TrimPrefix(path.Base(name), "/"), "/"), extension), nil
	} else {
		return fmt.Sprintf("%s/%s", filePath, strings.TrimPrefix(path.Base(name), "/")), nil
	}
}

func validPathMinio(path string) string {
	realPath := strings.TrimPrefix(path, "./")
	realPath = strings.TrimPrefix(realPath, "/")
	realPath = strings.TrimPrefix(realPath, ".")
	if realPath != "" && !strings.HasSuffix(realPath, "/") {
		realPath += "/"
	}

	return realPath
}
