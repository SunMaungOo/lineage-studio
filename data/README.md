# Introduction

The lineage studio is make up of ``repo`` of specific folder and files

# Components

1. metadata
2. links
3. obj

# Repo structure

```
  - root-folder-name
     - meta.json
     - links
         - objname.json
     - obj
         - objname-objhash.json
```

# Metadata

The metadata should be in root-folder-name and it should be name meta.json

```
{
  "type": "type of database",
  "host": "host location of server",
  "db": "database name"
}
```

# Object

The object is the view and procedure of the database. It should be under ``obj`` folder and the file name should be ``objname-objhash.json``

```
{
  "name": "name of the object",
  "detail": "view or procedure store code",
  "detailHash": "hash of the detail fail",
  "lineage": "lineage for the object",
  "lineageHash": "hash of the lineage",
  "createdAt": "UTC time the file was created",
  "verified": whether the lineage have been human verifed (true/false),
  "hash": "hash of detailHash + lineageHash which acts as hash of the object"
}
```

# Links

The purpose of the link file is to group the object file which have the same name. It should be under ``links`` folder and the the file name should be ``objname.json``

```
{
  "name": "name of the object",
  "type": "view/procedure",
  "current": "The current obj file hash",
  "history": [
    {
      "hash": "the obj file hash",
      // order of the obj file
      "order": 1
    },
    {
      "hash": "the obj file hash",
      "order": 2
    }
  ]
}
```

