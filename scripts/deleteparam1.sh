#

if [ $# -ne 2 ]
then
        echo "usage: $0 {param} file"
        exit 1
fi

a=$1
f=$2
# echo "Deleting param '$a' from '$f'"
n1=`wc -l < "$f"`
sed -e "/\"${a}.*\"/d" < "$f" > chall.tmp
cp chall.tmp "$f"
rm chall.tmp
n2=`wc -l < "$f"`
d=`expr $n1 - $n2`
if [ "$d" -ne 0 ]
then
	echo "Deleted param '$a' from '$f'"
fi
