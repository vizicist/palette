for i in P01 P02 P03 P04 ; do
	for k in C01 C02 C03 C04 C05 C06 C07 C08 C09 C10 C11 C12 C13 C14 C15 C16 ; do
		sed -e s/P01_C01/${i}_${k}/g < _template.json > ${i}_${k}.json 
	done
done

