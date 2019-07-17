package elasticSearch

const tlFolderWatchMapping = `{
  "settings": {
    "number_of_shards" : 1,
    "number_of_replicas" : 0,
    "analysis": {
      "analyzer": {
        "custom_path_tree": {
          "tokenizer": "custom_hierarchy"
        }
      },
      "tokenizer": {
        "custom_hierarchy": {
          "type": "path_hierarchy",
          "delimiter": "/"
        }
      }
    }
  },
  "mappings" : {
    "doc": {
      "properties" : {
        "name" : {
          "type" : "keyword"
        },
        "fullPath" : {
          "type" : "text",
          "fields": {
            "tree": {
              "type": "text",
              "analyzer": "custom_path_tree"
            },
            "keyword": {
              "type": "keyword"
            }
          }
        },
        "isDir" : {
          "type" : "boolean"
        },
        "isWatchFolder" : {
          "type" : "boolean"
        }
      }
    }
  }
}`
