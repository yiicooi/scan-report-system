package oss

import (
"fmt"
"time"

aliyunOSS "github.com/aliyun/aliyun-oss-go-sdk/oss"
"github.com/sui/scan-report/config"
)

var client *aliyunOSS.Client
var bucket *aliyunOSS.Bucket

func Init() error {
cfg := config.Cfg.OSS
if cfg.AccessKeyID == "" {
return nil
}
var err error
client, err = aliyunOSS.New(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret)
if err != nil {
return err
}
bucket, err = client.Bucket(cfg.BucketName)
return err
}

func PresignPutURL(objectKey string) (string, error) {
if bucket == nil {
return "", fmt.Errorf("OSS not configured")
}
return bucket.SignURL(objectKey, aliyunOSS.HTTPPut, int64(15*time.Minute/time.Second))
}

func ObjectURL(objectKey string) string {
return fmt.Sprintf("%s/%s", config.Cfg.OSS.Domain, objectKey)
}

func GenerateObjectKey(prefix, filename string) string {
return fmt.Sprintf("%s/%s/%d_%s", prefix, time.Now().Format("2006/01/02"), time.Now().UnixMilli(), filename)
}
