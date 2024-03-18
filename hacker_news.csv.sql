.headers on
.mode csv
.output hacker_news.csv
SELECT hn.id, hn.created_at, hn.time, hn.title, hn.url, hn.author, hn.ndescendants, hn.score, hn.rank
FROM (
	SELECT id, MAX(created_at) AS created_at FROM hn_items
	WHERE rank = 1 AND length(title) >= 5
	GROUP BY id
	ORDER BY time ASC
) t JOIN hn_items hn ON t.id = hn.id
ORDER BY hn.id, hn.created_at DESC;
