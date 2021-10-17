package fig

type Decoder interface {
	Decoder(interface{}) error
}
