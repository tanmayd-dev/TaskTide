package repositories

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/db"
)

func ensureBucketExists(ctx context.Context, minioClient *minio.Client, bucketName string) (error) {
	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		log.Printf("%v\n", err.Error())
		return errors.New("Could not check if bucket exists")
	}

	if !exists {
		err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			log.Printf("%v\n", err.Error())
			return errors.New("Could not create bucket")
		}
	}
	return nil
}

func UploadFile(queries *db.Queries, ctx context.Context, minioClient *minio.Client, bucketName string, contentType string, file io.Reader, fileName string, fileSize int64) (db.InputFile, error) {
	err := ensureBucketExists(ctx, minioClient, bucketName)
	if err != nil {
		return db.InputFile{}, err
	}

	minioFile, err := queries.CreateFile(
		ctx,
		fileName,
	)
	if err != nil {
		log.Printf("%v\n", err.Error())
		return db.InputFile{}, errors.New("Could not create file in database")
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
		return db.InputFile{}, errors.New("Could not add file to bucket")
	}
	return minioFile, nil
}

func GetFile(ctx context.Context, queries *db.Queries, file db.InputFile, minioClient *minio.Client, bucketName string) (*minio.Object, minio.ObjectInfo, error) {
	err := ensureBucketExists(ctx, minioClient, bucketName)
	if err != nil {
		return nil, minio.ObjectInfo{}, err
	}
	fileData, err := queries.GetFile(ctx, file.ID)
	if err != nil {
		log.Printf("%#v\n", err)
		return nil, minio.ObjectInfo{}, errors.New("Could not get fileData from db")
	}
	MinioID := fmt.Sprintf("%v/%v", file.ID, fileData.FileName)
	minioFile, err := minioClient.GetObject(
		ctx,
		bucketName,
		MinioID,
		minio.GetObjectOptions{},
	)
	if err != nil {
		log.Printf("%#v\n", err)
		return nil, minio.ObjectInfo{}, errors.New("Could not get object")
	}

	info, err := minioFile.Stat()
	if err != nil {
		log.Printf("%#v", err)
		return nil, minio.ObjectInfo{}, errors.New("Could not stat")
	}
	return minioFile, info, nil
}
