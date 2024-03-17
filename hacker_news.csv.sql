.headers on
.mode csv
.output hacker_news.csv
SELECT id, created_at, time, title, url, author, ndescendants, score, rank
FROM hn_items
WHERE id IN (SELECT hn_id FROM sn_items)
ORDER BY id, created_at DESC;
