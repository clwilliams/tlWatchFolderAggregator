package elasticSearch

const tlFolderWatchMapping = `{
  "settings": {
    "number_of_shards" : 1,
    "number_of_replicas" : 0
  },
  "mappings" : {
    "doc": {
      "properties" : {
        "name" : {
          "type" : "text",
          "index" : true
        },
        "path" : {
          "type" : "text",
          "index" : true
        },
        "isDir" : {
          "type" : "text",
          "index" : false
        },
        "isWatchFolder" : {
          "type" : "text",
          "index" : true
        }
      }
    }
  }
}`
