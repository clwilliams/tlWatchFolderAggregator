package elasticSearch

const tlFolderWatchMapping = `{
  "settings" : {
    "number_of_shards" : 1,
    "number_of_replicas" : 0
  },
  "mappings" : {
    "doc": {
      "properties" : {
        "name" : {
          "type" : "text",
          "index" : false
        },
        "isDir" : {
          "type" : "text",
          "index" : false
        },
        "path" : {
          "type" : "text",
          "index" : false
        }
      }
    }
  }
}`