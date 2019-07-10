## Brief summary

* tlWatchFolderAggregator - project for consuming message from Rabbit MQ & storing in elastic search. Also serves the API for the JSON calls.

* tlWatchFolder - project for watching a given folder and sending messages to RabbitMQ

* tlCommonMessaging - project for common RabbitMQ connection / setup and message body, used by both the above applications to save code duplication

All 3 need cloning to your machine and glide install ran for each.

## Getting everything up & running

You need to have the latest docker installed. Then

* Run up the docker in this project (provides RabbitMQ & elasticSearch)
```
docker-compose up
```
 - Local RabbitMQ  http://localhost:15672

 - ElasticSearch head  http://localhost:9100/

* Start tlWatchFolderAggregator (accept default args, but you can overwrite with environment variables / command line args if wanted), & run:
```
go run main.go
```

* Start tlWatchFolder (accept default args, again you can overwrite with environment variables / command line args if wanted), & run e.g.:
```
go run main.go --watchFolderPath="/Users/clairew/watch_me"
```

both of the above also take a --verbose flag if you want to see more logging on the console

## API

Get a JSON list of all the files and folders, ordered by path:
```
curl -X GET http://localhost:8000/all
```
Or retrieve for a specific watch folder:
```
curl -X GET http://localhost:8000/watch?folder=%2Fusers%2Fclairew%2Fwatch_me
```
