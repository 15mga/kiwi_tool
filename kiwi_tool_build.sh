DIR=`dirname $0`

OUTDIR=$DIR
PBDIR=$DIR
PBOUTDIR=$OUTDIR/../../
echo complie proto
protoc \
  --proto_path=$GOPATH/pb/ \
  --proto_path=$PBDIR  \
  --go_out=$PBOUTDIR \
  --csharp_out=$OUTDIR \
  $PBDIR/*.proto

echo finished