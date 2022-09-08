
synth=$1
name=$2

cp "Acoustic Grand Piano.json" "$name.json"
sed -e "s/001 Acoustic Grand Piano/$synth/" < "$name.json" > makesound.$$
mv makesound.$$ "$name.json"
