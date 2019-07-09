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
          "type" : "keyword"
        },
        "path" : {
          "type" : "text",
        },
        "isDir" : {
          "type" : "keyword"
        },
        "isWatchFolder" : {
          "type" : "keyword"
        }
      }
    }
  }
}`
