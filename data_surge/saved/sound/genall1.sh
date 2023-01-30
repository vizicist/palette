progoffset=0
for port in P1 P2 ; do
	for chan in C01 C02 C03 C04 C05 C06 C07 C08 C09 C10 C11 C12 C13 C14 C15 C16 ; do
		for prog in 1 2 3 4 5 6 7 8 ; do
			prognum=`expr $progoffset + $prog`
			echo ${port}${chan}_P${prognum}
			sed -e s/SYNTHNAME/${port}${chan}_P${prognum}/g < _template.json > ${port}${chan}_P${prognum}.json 
		done
		progoffset=`expr $progoffset + 8`
	done
done

