for port in P01 P02 P03 P04 ; do
	for chan in CH01 CH02 CH03 CH04 CH05 CH06 CH07 CH08 CH09 CH10 CH11 CH12 CH13 CH14 CH15 CH16 ; do
		echo ${port}_${chan}
		sed -e s/SYNTHNAME/${port}_${chan}/g < _template.json > ${port}_${chan}.json 
	done
done

