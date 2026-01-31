-- No reverse operation needed strictly, or revert to internal DNS
UPDATE authentic.media 
SET url = REPLACE(url, 'http://localhost:9000', 'minio-service:9000') 
WHERE url LIKE 'http://localhost:9000%';
