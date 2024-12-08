package main

import (
	// "fmt"
	"exame-final/mapreduce"
	"hash/fnv"

	"strconv"
	"strings"
	"unicode"
)

// mapFunc is called for each array of bytes read from the splitted files. For wordcount
// it should convert it into an array and parses it into an array of KeyValue that have
// all the words in the input.
func mapFunc(input []byte) (result []mapreduce.KeyValue) {
	var (
		text          string
		delimiterFunc func(c rune) bool
		words         []string
	)

	text = string(input)

	delimiterFunc = func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	}

	words = strings.FieldsFunc(text, delimiterFunc)

	// fmt.Printf("%v\n", words) //Para ajudar nos testes. Precisa da biblioteca fmt (acima comentada)

	result = make([]mapreduce.KeyValue, 0)

	for _, word := range words {
		//COMPLETAR ESSE CÓDIGO
		//Basta colocar em result os itens <word,"1">
		//Lembrando: word em minúsculo!
		result = append(result, mapreduce.KeyValue{Key: strings.ToLower(word), Value: "1"})
	}

	// fmt.Printf("%v\n", result) //Para ajudar nos testes. Precisa da biblioteca fmt (acima comentada)

	return result
}

// reduceFunc is called for each merged array of KeyValue resulted from all map jobs.
// It should return a similar array that summarizes all similar keys in the input.
func reduceFunc(input []mapreduce.KeyValue) (result []mapreduce.KeyValue) {
	// 	Maybe it's easier if you have an auxiliary structure:
	//      var mapAux map[string]int = make(map[string]int)
	//
	//  You need to do a loop in input
	// 	    for _,item := range input {
	//      ... }
	//
	//  You can check if a map have a key as following:
	// 	    _, ok := mapAux[item.Key]
	//  ok (true) means that the map has this key
	//  !ok means that the map does not have this key
	//
	// 	Reduce will receive KeyValue pairs (in variable input) that have string values, you may need
	// 	convert those values to int before being able to use it in operations.
	//  	package strconv: func Atoi(s string) (int, error)
	//
	//  However in result you need KeyValue pairs with strings.
	//	To convert int to string, use:
	//	package strconv: func Itoa(i int) string

	//COMPLETAR ESSE CÓDIGO!!!

	mapAux := make(map[string]int)

	for _, item := range input {
		_, ok := mapAux[item.Key]
		if ok {
			mapAux[item.Key]++
		} else {
			mapAux[item.Key] = 1
		}
	}

	result = make([]mapreduce.KeyValue, 0)

	for key, value := range mapAux {
		result = append(result, mapreduce.KeyValue{Key: key, Value: strconv.Itoa(value)})
	}

	// fmt.Printf("%v\n", result) //Para ajudar nos testes. Precisa da biblioteca fmt (acima comentada)

	return result
}

// shuffleFunc will shuffle map job results into different job tasks. It should assert that
// the related keys will be sent to the same job, thus it will hash the key (a word) and assert
// that the same hash always goes to the same reduce job.
// http://stackoverflow.com/questions/13582519/how-to-generate-hash-number-of-a-string-in-go
func shuffleFunc(task *mapreduce.Task, key string) (reduceJob int) {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() % uint32(task.NumReduceJobs))
}
