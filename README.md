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
Example response
```
[  
   {  
      "name":"watch_me",
      "isDir":"true",
      "fullPath":"/Users/clairew/watch_me",
      "isWatchFolder":"true"
   },
   {  
      "name":"2019",
      "isDir":"true",
      "fullPath":"/Users/clairew/watch_me/2019",
      "isWatchFolder":"false"
   },
   {  
      "name":"03 March",
      "isDir":"true",
      "fullPath":"/Users/clairew/watch_me/2019/03 March",
      "isWatchFolder":"false"
   },
   {  
      "name":"abc",
      "isDir":"true",
      "fullPath":"/Users/clairew/watch_me/2019/03 March/abc",
      "isWatchFolder":"false"
   },
   {  
      "name":"de Gournay Chinoiserie C076 Chatsworth.pdf",
      "isDir":"false",
      "fullPath":"/Users/clairew/watch_me/2019/03 March/de Gournay Chinoiserie C076 Chatsworth.pdf",
      "isWatchFolder":"false"
   },
   {  
      "name":"04 April",
      "isDir":"true",
      "fullPath":"/Users/clairew/watch_me/2019/04 April",
      "isWatchFolder":"false"
   },
   {  
      "name":"05 May",
      "isDir":"true",
      "fullPath":"/Users/clairew/watch_me/2019/04 April/05 May",
      "isWatchFolder":"false"
   },
   {  
      "name":"de Gournay Chinoiserie  C078 Badminton.pdf",
      "isDir":"false",
      "fullPath":"/Users/clairew/watch_me/2019/04 April/de Gournay Chinoiserie  C078 Badminton.pdf",
      "isWatchFolder":"false"
   },
   {  
      "name":"de Gournay Chinoiserie C076 Chatsworth.pdf",
      "isDir":"false",
      "fullPath":"/Users/clairew/watch_me/2019/04 April/de Gournay Chinoiserie C076 Chatsworth.pdf",
      "isWatchFolder":"false"
   },
   {  
      "name":"de Gournay Chinoiserie cC075 Jardinières \u0026 Citrus Trees.pdf",
      "isDir":"false",
      "fullPath":"/Users/clairew/watch_me/2019/04 April/de Gournay Chinoiserie cC075 Jardinières \u0026 Citrus Trees.pdf",
      "isWatchFolder":"false"
   }
]
```

Or retrieve for a specific watch folder:
```
curl -X GET http://localhost:8000/watch?folder=%2FUsers%2Fclairew%2Fwatch_me%2F2019%2F03
```
example response:
```
[  
   {  
      "name":"03 March",
      "isDir":"true",
      "fullPath":"/Users/clairew/watch_me/2019/03 March",
      "isWatchFolder":"false"
   },
   {  
      "name":"abc",
      "isDir":"true",
      "fullPath":"/Users/clairew/watch_me/2019/03 March/abc",
      "isWatchFolder":"false"
   },
   {  
      "name":"de Gournay Chinoiserie C076 Chatsworth.pdf",
      "isDir":"false",
      "fullPath":"/Users/clairew/watch_me/2019/03 March/de Gournay Chinoiserie C076 Chatsworth.pdf",
      "isWatchFolder":"false"
   }
]
```
