build:
	go build -o scraper_scheduler
	cd executor && go build -o scraper_executor

run: build
	ACCESS_KEY=<CHANGE ME> SECRET_KEY=<CHANGE ME> ./scraper_scheduler --master=127.0.0.1:5050 --executor="/home/vagrant/go/src/github.com/mesosphere/mesos-framework-tutorial/executor/scraper_executor" --logtostderr=true
