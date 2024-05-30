package codecs

type PayloadDescriptor interface {
	Dump()
}

type EncodingContextParams struct {
	SpatialLayers  uint8
	TemporalLayers uint8
	Ksvc           bool
}

type EncodingContext struct {
	params               EncodingContextParams
	targetSpatialLayer   int16
	targetTemporalLayer  int16
	currentSpatialLayer  int16
	currentTemporalLayer int16
	ignoreDtx            bool
}

func NewEncodingContext(params EncodingContextParams) *EncodingContext {
	return &EncodingContext{
		params:               params,
		targetSpatialLayer:   -1,
		targetTemporalLayer:  -1,
		currentSpatialLayer:  -1,
		currentTemporalLayer: -1,
	}
}

func (ec *EncodingContext) GetSpatialLayers() uint8 {
	return ec.params.SpatialLayers
}

func (ec *EncodingContext) GetTemporalLayers() uint8 {
	return ec.params.TemporalLayers
}

func (ec *EncodingContext) IsKSvc() bool {
	return ec.params.Ksvc
}

func (ec *EncodingContext) GetTargetSpatialLayer() int16 {
	return ec.targetSpatialLayer
}

func (ec *EncodingContext) GetTargetTemporalLayer() int16 {
	return ec.targetTemporalLayer
}

func (ec *EncodingContext) GetCurrentSpatialLayer() int16 {
	return ec.currentSpatialLayer
}

func (ec *EncodingContext) GetCurrentTemporalLayer() int16 {
	return ec.currentTemporalLayer
}

func (ec *EncodingContext) GetIgnoreDtx() bool {
	return ec.ignoreDtx
}

func (ec *EncodingContext) SetTargetSpatialLayer(spatialLayer int16) {
	ec.targetSpatialLayer = spatialLayer
}

func (ec *EncodingContext) SetTargetTemporalLayer(temporalLayer int16) {
	ec.targetTemporalLayer = temporalLayer
}

func (ec *EncodingContext) SetCurrentSpatialLayer(spatialLayer int16) {
	ec.currentSpatialLayer = spatialLayer
}

func (ec *EncodingContext) SetCurrentTemporalLayer(temporalLayer int16) {
	ec.currentTemporalLayer = temporalLayer
}

func (ec *EncodingContext) SetIgnoreDtx(ignore bool) {
	ec.ignoreDtx = ignore
}

type PayloadDescriptorHandler interface {
	Dump()
	Process(context *EncodingContext, data []byte) (marker, ok bool)
	Restore(data []byte)
	GetSpatialLayer() uint8
	GetTemporalLayer() uint8
	IsKeyFrame() bool
}
