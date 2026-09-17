SELECT * 
FROM kaskel_transaction 
WHERE created_date > ? 
ORDER BY created_date ASC;