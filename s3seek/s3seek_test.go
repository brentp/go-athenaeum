package s3seek_test

import (
	"bufio"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/brentp/go-athenaeum/s3seek"
)

func TestRead(t *testing.T) {
	sess := session.Must(session.NewSession())
	svc := s3.New(sess, aws.NewConfig().WithRegion("us-east-1"))

	s, err := s3seek.New(svc, "s3://1000genomes/20131219.populations.tsv", nil)
	if err != nil {
		t.Fatal(err)
	}

	r := bufio.NewReader(s)
	l1, err := r.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	l2, err := r.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	_, err = s.Seek(int64(len(l1)), 0)
	if err != nil {
		t.Fatal(err)
	}

	r = bufio.NewReader(s)
	l22, err := r.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}

	if l2 != l22 {
		t.Fatal("error seeking. expected line 2 from test file")
	}

}

/*
func TestBam(t *testing.T) {
	sess := session.Must(session.NewSession())
	svc := s3.New(sess, aws.NewConfig().WithRegion("us-east-1"))

	s, err := s3seek.New(svc, "s3://1000genomes/phase3/data/NA12878/high_coverage_alignment/NA12878.mapped.ILLUMINA.bwa.CEU.high_coverage_pcr_free.20130906.bam.bai", nil)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := bam.ReadIndex(s)
	if err != nil {
		t.Fatal(err)
	}

	s.Close()

	s, err = s3seek.New(svc, "s3://1000genomes/phase3/data/NA12878/high_coverage_alignment/NA12878.mapped.ILLUMINA.bwa.CEU.high_coverage_pcr_free.20130906.bam", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	b, err := bam.NewReader(s, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	ref := b.Header().Refs()[4]

	chunks, err := idx.Chunks(ref, 45000000, 45000100)
	if err != nil {
		t.Fatal(err)
	}

	bi, err := bam.NewIterator(b, chunks)

	for bi.Next() {
		rec := bi.Record()
		fmt.Println(rec.Ref.Name(), rec.Start())
	}
	if err = bi.Error(); err != nil {
		t.Fatal(err)
	}

}
*/
