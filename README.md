# Usage

The `natsstore` package provides a NATS-based session store implementation for use with Gorilla sessions. Here's how to use it in your Go application:

## Installation

First, install the package:

```
go get github.com/hugozap/natsstore
```

## Importing

Import the package in your Go code:

```go
import "github.com/hugozap/natsstore"
```

## Creating a New Store

To create a new NATS session store, you need a NATS JetStream context and a key prefix:

```go
//js is a nats.JetStreamContext
// Create a new store with a key prefix and optional key pairs for encryption
store, err := natsstore.NewStore(js, "myapp", []byte("secret-key"))
if err != nil {
    // Handle error
}
```

## Using the Store with Gorilla Sessions

Once you have created the store, you can use it with Gorilla sessions:

```go
// Create a new session
session, err := store.New(r, "session-name")
if err != nil {
    // Handle error
}

// Get an existing session
session, err := store.Get(r, "session-name")
if err != nil {
    // Handle error
}

// Set a value in the session
session.Values["user_id"] = "123"

// Save the session
err = store.Save(r, w, session)
if err != nil {
    // Handle error
}
```



