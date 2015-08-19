build:
	go build -o whampire_scheduler
	cd executor && go build -o whampire_executor

run: build
	ACCESS_KEY=$(ACCESS_KEY) SECRET_KEY=$(SECRET_KEY) ./whampire_scheduler --master=127.0.0.1:5050 --executor="./executor/whampire_executor" --logtostderr=true
