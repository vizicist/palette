echo "{ \"synths\" : ["
for port in 01 02 03 04; do
	for chan in 01 02 03 04 05 06 07 08 09 10 11 12 13 14 15 16 ; do
		chnum=`echo ${chan} | sed -e 's/^0//'`
		echo "    {\"name\": \"P${port}_CH${chan}\", \"port\":\"${port}. Internal MIDI\", \"channel\":${chnum}},"
		sep=","
	done
done
echo "    {\"name\": \"default\", \"port\":\"01. Internal MIDI\", \"channel\":1}"
echo "]"
echo "}"
