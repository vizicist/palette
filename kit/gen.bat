go generate
sed -e '/\/\/line .*:/d' < gram.go > newgram.go
copy newgram.go gram.go
del newgram.go
