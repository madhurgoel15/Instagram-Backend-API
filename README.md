# Instagram-Backend-API 
Run main.go for the program<br>
Only the standard packages provided by go have been used in this program.<br>
Here we are using switch case to differentiate between the type of query by user. Then depending on query, we connect to the database and call the respective function of the query. After the query is completed, we call the close function to close the connection to the database. Also, when we call the connect function, we lock the mutex so, only one connection to the database is made at a given time to protect the system from deadlock or data inconsistency. After closing the connection to the database, we unlock the mutex to release the database for further queries.<br>
When entering data for creating user and post, json data needs to be passed in the request body.<br>
In the find all posts of a given user_id query, pagination is used. Default page size is 20. To navigate through different pages, enter query using the format users/posts/user_id?page=page_no<br>
The encryption function used to protect the password is AES CYPHER.<br>
We use a random key, then generate its hash to get a 32-bit key. Now we use this key to Encrypt the password using AES Cypher.<br>
The timestamp format used here is standard ISO.<br>
Database used is MongoDB.<br>
