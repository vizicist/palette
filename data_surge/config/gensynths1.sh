progoffset=0
echo "{ \"synths\" : ["
for port in 1 2 ; do
	for chan in 01 02 03 04 05 06 07 08 09 10 11 12 13 14 15 16 ; do
		for prog in 1 2 3 4 5 6 7 8 ; do
			prognum=`expr $progoffset + $prog`
			chnum=`echo ${chan} | sed -e 's/^0//'`
			echo "    {\"name\": \"P${port}C${chan}_P${prognum}\", \"port\":\"0${port}. Internal MIDI\", \"channel\":${chnum}, \"program\":${prognum}},"
			sep=","
		done
		progoffset=`expr $progoffset + 8`
	done
done
echo "    {\"name\": \"default\", \"port\":\"01. Internal MIDI\", \"channel\":1, \"program\":1}"
echo "]"
echo "}"
