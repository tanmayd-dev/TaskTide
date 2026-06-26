package repositories

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/utils"
)

func ensureBucketExists(ctx context.Context, minioClient *minio.Client, bucketName string) error {
	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		log.Printf("%v\n", err.Error())
		return errors.New("could not check if bucket exists")
	}

	if !exists {
		err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Printf("%v\n", err.Error())
			return errors.New("could not create bucket")
		}
	}
	return nil
}

func UploadFile(queries *db.Queries, ctx context.Context, minioClient *minio.Client, bucketName string, contentType string, file io.Reader, fileName string, fileSize int64) (models.InputFile, error) {
	err := ensureBucketExists(ctx, minioClient, bucketName)
	if err != nil {
		return models.InputFile{}, err
	}

	minioFile, err := queries.CreateFile(
		ctx,
		fileName,
	)
	if err != nil {
		log.Printf("%v\n", err.Error())
		return models.InputFile{}, errors.New("could not create file in database")
	}

	_, err = minioClient.PutObject(
		ctx,
		bucketName,
		fmt.Sprintf("%v/%v", minioFile.ID, fileName),
		file,
		fileSize,
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)
	if err != nil {
		log.Printf("%v\n", err.Error())
		err := queries.DeleteFile(ctx, minioFile.ID)
		if err != nil {
			log.Printf("%v\n", err.Error())
		}
		return models.InputFile{}, errors.New("could not add file to bucket")
	}
	return utils.InputFileModel(minioFile), nil
}

func CreateDefaultNullFile(ctx context.Context, queries *db.Queries, minioClient *minio.Client, bucketName string) (models.InputFile, error) {
	return CreateEmptyFile(ctx, queries, minioClient, bucketName, "default-null-file")
}

func CreateEmptyFile(ctx context.Context, queries *db.Queries, minioClient *minio.Client, bucketName, fileName string) (models.InputFile, error) {
	err := ensureBucketExists(ctx, minioClient, bucketName)
	if err != nil {
		return models.InputFile{}, err
	}

	minioFile, err := queries.CreateFile(ctx, fileName)
	if err != nil {
		log.Printf("%v\n", err.Error())
		return models.InputFile{}, errors.New("could not create file in database")
	}

	_, err = minioClient.PutObject(
		ctx,
		bucketName,
		fmt.Sprintf("%v/%v", minioFile.ID, minioFile.FileName),
		bytes.NewReader([]byte{}),
		0,
		minio.PutObjectOptions{
			ContentType: "application/octet-stream",
		},
	)
	if err != nil {
		log.Printf("%v\n", err.Error())
		deleteErr := queries.DeleteFile(ctx, minioFile.ID)
		if deleteErr != nil {
			log.Printf("%v\n", deleteErr.Error())
		}
		return models.InputFile{}, errors.New("could not add file to bucket")
	}
	return utils.InputFileModel(minioFile), nil
}

func GetFile(ctx context.Context, queries *db.Queries, fileID models.InputFileID, minioClient *minio.Client, bucketName string) (*minio.Object, minio.ObjectInfo, error) {
	err := ensureBucketExists(ctx, minioClient, bucketName)
	if err != nil {
		return nil, minio.ObjectInfo{}, err
	}
	fileUUID, err := utils.StringToUUID(string(fileID))
	if err != nil {
		return nil, minio.ObjectInfo{}, err
	}
	fileData, err := queries.GetFile(ctx, fileUUID)
	if err != nil {
		log.Printf("%#v\n", err)
		return nil, minio.ObjectInfo{}, errors.New("could not get fileData from db")
	}
	MinioID := fmt.Sprintf("%v/%v", fileID, fileData.FileName)
	minioFile, err := minioClient.GetObject(
		ctx,
		bucketName,
		MinioID,
		minio.GetObjectOptions{},
	)
	if err != nil {
		log.Printf("%#v\n", err)
		return nil, minio.ObjectInfo{}, errors.New("could not get object")
	}

	info, err := minioFile.Stat()
	if err != nil {
		log.Printf("%#v", err)
		return nil, minio.ObjectInfo{}, errors.New("could not stat")
	}
	return minioFile, info, nil
}

func UpdateFileContent(ctx context.Context, queries *db.Queries, minioClient *minio.Client, bucketName string, fileID models.InputFileID, content string) error {
	err := ensureBucketExists(ctx, minioClient, bucketName)
	if err != nil {
		return err
	}
	fileUUID, err := utils.StringToUUID(string(fileID))
	if err != nil {
		return err
	}
	fileData, err := queries.GetFile(ctx, fileUUID)
	if err != nil {
		return err
	}
	MinioID := fmt.Sprintf("%v/%v", fileID, fileData.FileName)
	_, err = minioClient.PutObject(
		ctx,
		bucketName,
		MinioID,
		bytes.NewReader([]byte(content)),
		int64(len(content)),
		minio.PutObjectOptions{
			ContentType: "text/plain",
		},
	)
	return err
}

