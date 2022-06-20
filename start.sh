screen -r paint -X stuff "^C\n"
sleep 10s
screen -r paint -X kill
screen -U -A -md -S paint
screen -r paint -X stuff "cd $(dirname $0)/\n"
screen -r paint -X stuff "while :; do go run main.go ; done\n"
