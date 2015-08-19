# Whampire

This repo is an example Mesos Framework built at the MesosCon 2015 Mesos Hackathon.

![logo](whampire.png)

The framework comes with a scheduler and executor that work together to scrape web urls.

Our goals were to:

 - Learn the basics of Mesos Framework internals
 - Utilize an API in the scheduler
 - Integrate a database for persistant storage

## Building

Build the scheduler:

```bash
$ go build -o whampire_scheduler
```

Build the executor:

```bash
$ cd executor
$ go build -o whampire_scheduler
```

## Running

```bash
$ export ACCESS_KEY=<CHANGEME>
$ export SECRET_KEY=<CHANGEME>
$ ./whampire_scheduler --master=127.0.0.1:5050 --executor="./executor/whampire_executor" --logtostderr=true
```
