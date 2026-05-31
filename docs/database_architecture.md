## Database layer architecture 

**Data ownership**
- Database own collection
- Collection own index and wal 
- Index own Vector
- Vector own embeddings values
- Wal own segement files

**Database layer responsiblities**
- store collection or multiple collections 
- manage & orchestrate collection lifecycle

**Database invariants**
- each database has unique name and database identity is owned by name
- a single database can have multiple unique collections
- database registry only holds currently opened collections including newly created  not all the collection from disk
- closing a database will close all the collections of the database
- database can have 0 collection, database with 0 collection is empty
- a collection of database can be opened once only, opening same collectin give ErrCollectionAlreadyOpen
- database can delete multiple collection (idempotent)
- database can delete collection permanently (future product-feature sub trash system, soft-delete and roll back)
- database can create multiple collections
- database doesn't not handle wal/index/vector, only collection

*Database Representation*
>   struct Database {
        Name : string -> unique name of the Database
        RootDir : sting  ->  root path to the database directory
        collections : map[string]*Collection -> list of the collections in database
   } 

**Database Core API**
- CreateDatabase("database_name" string, rootDir string) (*Database, error)
- OpenDatabase("database_name", rootDir string) (*Database, error)
- database_name.CreateCollection(collection_config collection.CollectionConfig, wal.SynciPolicy) (error)
- database_name.OpenCollection("collection_name" string, wal.SyncPolicy) (*collection.Collection, error)
- database_name.GetCollection("collection_name" string) (*collection.Collection, error)
- database_name.CloseCollection("collection_name" string) (error)
- database_name.DeleteCollection("collection_name" string) (error)
- database_name.Close() (error)
- (future) db.RestoreCollection("collection_name") (error)
- database_name.List() ([]string "collection in db") -> (future create a metadata.json to store list of collections/etc)
rootDir/
 └── dbName/
      └── metadata.json