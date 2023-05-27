# curl
curl -X POST http://0.0.0.0:8000/set
   -H 'Content-Type: application/json'
   -d '{"key":"test-key", "value":"test-value"}'

curl http://0.0.0.0:8000/get?key=test-key

curl -X POST http://0.0.0.0:8000/update
   -H 'Content-Type: application/json'
   -d '{"key":"test-key", "value":"updated-value"}'

curl http://0.0.0.0:8000/delete?key=test-key


# HTTPie
http POST :8000/set key=test-key value=test-key

http GET :8000/get key==test-key

http POST :8000/update key=test-key value=updated-key

http DELETE :8000/delete key==test-key
