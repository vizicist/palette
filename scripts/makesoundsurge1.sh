
synth=$1
name=$2

cp "0101 Fantasy_Bell.json" "$name.json"
sed -e "s/0[0-9]01 Fantasy_Bell/$synth/" < "$name.json" > makesound.$$
mv makesound.$$ "$name.json"
