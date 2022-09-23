if [ $# -ne 1 ]
then
	echo "Usage: $0 {param}"
	exit 1
fi
find . -name "*.json" ! -name _override.json -print0 | xargs -n 1 -0 deleteparam1.sh "$1"
