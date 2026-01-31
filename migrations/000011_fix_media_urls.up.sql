UPDATE authentic.media 
SET url = REPLACE(url, 'minio-service:9000', 'http://localhost:9000') 
WHERE url LIKE 'minio-service:9000%';
