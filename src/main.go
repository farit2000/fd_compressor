package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/farit2000/compressor/src/bwt"
	"github.com/farit2000/compressor/src/huffman"
	"github.com/farit2000/compressor/src/mtf"
	"github.com/farit2000/compressor/src/rle"
	"io/ioutil"
	"log"
	"os"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// Метод компрессии, в котором используется 4 этапа сжатия BWTs -> RLE -> NTF -> Huffman
func compress(inputFilePath string, outPutFilePath string) error {
	normBytes, err := ioutil.ReadFile(inputFilePath)

	//get bwt bytes
	size := uint(len(normBytes))
	bwtBytes := make([]byte, size)
	bwtComp, err := bwt.NewBWTS()
	if err != nil {
		return err
	}
	_, _, err = bwtComp.Forward(normBytes, bwtBytes)
	check(err)

	//get rle bytes
	rleString := []byte(rle.RunLengthEncode(string(bwtBytes)))

	//get mtf bytes
	alphabet := mtf.AlphabetCreate(rleString)
	m := mtf.SymbolTable(alphabet)
	mtfBytes := m.Encode(rleString)
	mtfBytes = append(mtfBytes, alphabet...)
	mtfBytes = append(mtfBytes, byte(len(alphabet)))

	//write huffman bites
	f, err := os.Create(outPutFilePath)
	check(err)
	w := huffman.NewWriter(f)
	if _, err := w.Write(mtfBytes); err != nil {
		log.Panicln("Failed to write:", err)
	}
	if err := w.Close(); err != nil {
		log.Panicln("Failed to close:", err)
	}
	defer f.Close()
	return nil
}

// Метод декомпрессии, все происходит в обратном порядке
func decompress(inputFilePath string, outPutFilePath string) error {
	f, err := os.Open(inputFilePath)
	check(err)

	//get huffman bytes
	r := huffman.NewReader(f)
	defer f.Close()
	mtfBytes, err := ioutil.ReadAll(r)
	check(err)

	//get mtf bytes
	mtfBytes, alphabet := mtf.GetAlphabet(mtfBytes)
	m := mtf.SymbolTable(alphabet)

	//get rle bytes
	rleBytes := m.Decode(mtfBytes)
	//bwtBytes := m.Decode(mtfBytes)

	//get bwt bytes
	bwtBytes := []byte(rle.RunLengthDecode(string(rleBytes)))

	//get norm bytes
	size := uint(len(bwtBytes))
	normBytes := make([]byte, size)
	bwtComp, err := bwt.NewBWTS()
	if err != nil {
		return err
	}
	_, _, err = bwtComp.Inverse(bwtBytes, normBytes)
	check(err)
	_, _, err = bwtComp.Inverse(bwtBytes, normBytes)
	check(err)

	err = ioutil.WriteFile(outPutFilePath, normBytes, 0777)
	check(err)
	return nil
}


func main() {
	inputFilePath := flag.String("i", "", "a string")
	outputFilePath := flag.String("o", "", "a string")
	flag.Parse()
	if *inputFilePath == "" {
		panic(errors.New("inputFile path in empty"))
	}
	if *outputFilePath == "" {
		panic(errors.New("outputFilePath path in empty"))
	}

	//compressor
	//err := compress(*inputFilePath, *outputFilePath)
	//if err != nil {
	//	fmt.Printf("Error while compressing %s", err.Error())
	//	panic(err)
	//}
	//fmt.Printf("Compress successful. Compressed file path is %s\n", *outputFilePath)

	//decompressor
	err := decompress(*inputFilePath, *outputFilePath)
	if err != nil {
		fmt.Printf("Error while decompressing %s", err.Error())
		panic(err)
	}
	fmt.Printf("Decompress successful. Decompressed file path is %s\n", *outputFilePath)


	// код для удобства тестирования

	//action := os.Args[1]
	//inputFilePath := os.Args[2]
	//outputFilePath := os.Args[3]
	//
	//action := "compress"
	//inputFilePath :=  "/Users/faritshamardanov/Downloads/waputf8.txt"
	//outputFilePath := "/Users/faritshamardanov/go/src/github.com/farit2000/compressor/testData/compress_result_b.fd"
	//
	//action := "decompress"
	//inputFilePath :=  "/Users/faritshamardanov/go/src/github.com/farit2000/compressor/testData/compress_result_b.fd"
	//outputFilePath := "/Users/faritshamardanov/go/src/github.com/farit2000/compressor/testData/decompress_res.txt"
	//
	//switch action {
	//case "compress":
	//	err := compress(inputFilePath, outputFilePath)
	//	if err != nil {
	//		fmt.Printf("Error while compressing %s", err.Error())
	//		panic(err)
	//	}
	//	fmt.Printf("Compress successful. Compressed file path is %s", outputFilePath)
	//case "decompress":
	//	err := decompress(inputFilePath, outputFilePath)
	//	if err != nil {
	//		fmt.Printf("Error while decompressing %s", err.Error())
	//		panic(err)
	//	}
	//	fmt.Printf("Decompress successful. Decompressed file path is %s", outputFilePath)
	//default:
	//	fmt.Printf("Command %s unsupported", action)
	//}
}