GOOS=linux GOARCH=amd64 go build -o food-delivery-notifier_linux github.com/romanthekat/food-delivery-notifier && \
GOOS=darwin GOARCH=amd64 go build -o food-delivery-notifier_mac github.com/romanthekat/food-delivery-notifier && \
GOOS=windows GOARCH=amd64 go build -o food-delivery-notifier_win github.com/romanthekat/food-delivery-notifier