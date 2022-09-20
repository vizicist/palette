#

if [ $# -ne 3 ]
then
        echo "usage: $0 {string1} {string2} file"
        exit 1
fi

a=$1
b=$2
f=$3
shift
shift
echo "Replacing '$a' with '$b'"
echo "** $f ** is being changed to:"
sed -n -e "s${a}${b}gp" < "$f"
sed -e "s${a}${b}g" < "$f" > chall.tmp
cp chall.tmp "$f"
rm chall.tmp
