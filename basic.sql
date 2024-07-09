SELECT DISTINCT name FROM media
ORDER BY size DESC;

SELECT  * FROM media
WHERE name = 'IMG_0060.jpg'
ORDER BY size DESC;

SELECT * FROM media
ORDER BY size DESC;

SELECT * FROM media
WHERE mime_type NOT IN ('video', 'audio')
ORDER BY size DESC;

SELECT DISTINCT mime_type FROM media;


SELECT * FROM media
WHERE mime_type  IN ('image')
ORDER BY size DESC;