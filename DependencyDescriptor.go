package DependencyDescriptor

type DependencyDescriptor struct {
	data                                       []byte
	pos                                        int
	start_of_frame                             int
	end_of_frame                               int
	frame_number                               int
	frame_dependency_template_id               int
	template_dependency_structure_present_flag int
	active_decode_targets_present_flag         int
	custom_dtis_flag                           int
	custom_fdiffs_flag                         int
	custom_chains_flag                         int
	active_decode_targets_bitmask              int
	template_id_offset                         int
	dt_cnt_minus_one                           int
	resolutions_present_flag                   int
	next_layer_idc                             int
	DtCnt                                      int
	TemplateCnt                                int
	MaxTemporalId                              int
	MaxSpatialId                               int
	chain_cnt                                  int
	FrameSpatialId                             int
	FrameTemporalId                            int
	FrameFdiffCnt                              int
	FrameMaxWidth                              int
	FrameMaxHeight                             int
	TotalConsumedBits                          int
	zero_padding                               int

	frame_chain_fdiff          map[int]int
	TemplateSpatialId          map[int]int
	TemplateTemporalId         map[int]int
	TemplateFdiffCnt           map[int]int
	FrameFdiff                 map[int]int
	frame_dti                  map[int]int
	max_render_width_minus_1   map[int]int
	max_render_height_minus_1  map[int]int
	DecodeTargetSpatialId      map[int]int
	DecodeTargetTemporalId     map[int]int
	decode_target_protected_by map[int]int
	template_dti               map[int]map[int]int
	TemplateFdiff              map[int]map[int]int
	template_chain_fdiff       map[int]map[int]int
}

func NewDependencyDescriptor() *DependencyDescriptor {
	return &DependencyDescriptor{
		frame_chain_fdiff:          make(map[int]int),
		TemplateSpatialId:          make(map[int]int),
		TemplateTemporalId:         make(map[int]int),
		TemplateFdiffCnt:           make(map[int]int),
		FrameFdiff:                 make(map[int]int),
		frame_dti:                  make(map[int]int),
		max_render_width_minus_1:   make(map[int]int),
		max_render_height_minus_1:  make(map[int]int),
		DecodeTargetSpatialId:      make(map[int]int),
		DecodeTargetTemporalId:     make(map[int]int),
		decode_target_protected_by: make(map[int]int),
		template_dti:               make(map[int]map[int]int),
		TemplateFdiff:              make(map[int]map[int]int),
		template_chain_fdiff:       make(map[int]map[int]int),
	}
}

func (p *DependencyDescriptor) Unmarshal(d []byte) {
	p.data = d
	p.pos = 0
	p.dependency_descriptor(len(d))
}

func (p *DependencyDescriptor) f(n int) int {
	x := 0
	for i := 0; i < n; i++ {
		x = 2*x + int(p.read_bit())
	}
	p.TotalConsumedBits += n
	return x
}

func (p *DependencyDescriptor) read_bit() uint8 {
	x := (p.data[p.pos/8] >> (7 - (p.pos % 8))) & 1
	p.pos++
	return x
}

func (p *DependencyDescriptor) ns(n int) int {
	w := 0
	x := n
	for x != 0 {
		x = x >> 1
		w++
	}
	m := (1 << w) - n
	v := p.f(w - 1)
	if v < m {
		return v
	}
	extra_bit := p.f(1)
	return (v << 1) - m + extra_bit
}

func (p *DependencyDescriptor) dependency_descriptor(sz int) {
	p.TotalConsumedBits = 0
	p.mandatory_descriptor_fields()
	if sz > 3 {
		p.extended_descriptor_fields()
	} else {
		p.no_extended_descriptor_fields()
	}
	p.frame_dependency_definition()
	p.zero_padding = p.f(sz*8 - p.TotalConsumedBits)
}

func (p *DependencyDescriptor) mandatory_descriptor_fields() {
	p.start_of_frame = p.f(1)
	p.end_of_frame = p.f(1)
	p.frame_dependency_template_id = p.f(6)
	p.frame_number = p.f(16)
}

func (p *DependencyDescriptor) extended_descriptor_fields() {
	p.template_dependency_structure_present_flag = p.f(1)
	p.active_decode_targets_present_flag = p.f(1)
	p.custom_dtis_flag = p.f(1)
	p.custom_fdiffs_flag = p.f(1)
	p.custom_chains_flag = p.f(1)

	if p.template_dependency_structure_present_flag == 1 {
		p.template_dependency_structure()
		p.active_decode_targets_bitmask = (1 << p.DtCnt) - 1
	}

	if p.active_decode_targets_present_flag == 1 {
		p.active_decode_targets_bitmask = p.f(int(p.DtCnt))
	}
}

func (p *DependencyDescriptor) no_extended_descriptor_fields() {
	p.custom_dtis_flag = 0
	p.custom_fdiffs_flag = 0
	p.custom_chains_flag = 0
}

func (p *DependencyDescriptor) template_dependency_structure() {
	p.template_id_offset = p.f(6)
	p.dt_cnt_minus_one = p.f(5)
	p.DtCnt = p.dt_cnt_minus_one + 1
	p.template_layers()
	p.template_dtis()
	p.template_fdiffs()
	p.template_chains()
	p.decode_target_layers()
	p.resolutions_present_flag = p.f(1)
	if p.resolutions_present_flag == 1 {
		p.render_resolutions()
	}
}

func (p *DependencyDescriptor) frame_dependency_definition() {
	templateIndex := (p.frame_dependency_template_id + 64 - p.template_id_offset) % 64
	if templateIndex >= p.TemplateCnt {
		return // error
	}
	p.FrameSpatialId = p.TemplateSpatialId[templateIndex]
	p.FrameTemporalId = p.TemplateTemporalId[templateIndex]

	if p.custom_dtis_flag == 1 {
		p.frame_dtis()
	} else {
		p.frame_dti = p.template_dti[templateIndex]
	}

	if p.custom_fdiffs_flag == 1 {
		p.frame_fdiffs()
	} else {
		p.FrameFdiffCnt = p.TemplateFdiffCnt[templateIndex]
		p.FrameFdiff = p.TemplateFdiff[templateIndex]
	}

	if p.custom_chains_flag == 1 {
		p.frame_chains()
	} else {
		p.frame_chain_fdiff = p.template_chain_fdiff[templateIndex]
	}

	if p.resolutions_present_flag == 1 {
		p.FrameMaxWidth = p.max_render_width_minus_1[p.FrameSpatialId] + 1
		p.FrameMaxHeight = p.max_render_height_minus_1[p.FrameSpatialId] + 1
	}
}

func (p *DependencyDescriptor) template_layers() {
	temporalId := 0
	spatialId := 0
	p.TemplateCnt = 0
	p.MaxTemporalId = 0
	for {
		p.TemplateSpatialId[p.TemplateCnt] = spatialId
		p.TemplateTemporalId[p.TemplateCnt] = temporalId
		p.TemplateCnt++
		p.next_layer_idc = p.f(2)
		if p.next_layer_idc == 1 {
			temporalId++
			if temporalId > p.MaxTemporalId {
				p.MaxTemporalId = temporalId
			}
		} else if p.next_layer_idc == 2 {
			temporalId = 0
			spatialId++
		}
		if p.next_layer_idc == 3 {
			break
		}
	}
	p.MaxSpatialId = spatialId
}

func (p *DependencyDescriptor) render_resolutions() {
	for spatial_id := 0; spatial_id <= p.MaxSpatialId; spatial_id++ {
		p.max_render_width_minus_1[spatial_id] = p.f(16)
		p.max_render_height_minus_1[spatial_id] = p.f(16)
	}
}

func (p *DependencyDescriptor) template_dtis() {
	for templateIndex := 0; templateIndex < p.TemplateCnt; templateIndex++ {
		p.template_dti[templateIndex] = make(map[int]int)
		for dtIndex := 0; dtIndex < p.DtCnt; dtIndex++ {
			p.template_dti[templateIndex][dtIndex] = p.f(2)
		}
	}
}

func (p *DependencyDescriptor) frame_dtis() {
	for dtIndex := 0; dtIndex < p.DtCnt; dtIndex++ {
		p.frame_dti[dtIndex] = p.f(2)
	}
}

func (p *DependencyDescriptor) template_fdiffs() {
	for templateIndex := 0; templateIndex < p.TemplateCnt; templateIndex++ {
		fdiffCnt := 0
		fdiff_follows_flag := p.f(1)
		p.TemplateFdiff[templateIndex] = make(map[int]int)
		for fdiff_follows_flag == 1 {
			fdiff_minus_one := p.f(4)
			p.TemplateFdiff[templateIndex][fdiffCnt] = fdiff_minus_one + 1
			fdiffCnt++
			fdiff_follows_flag = p.f(1)
		}
		p.TemplateFdiffCnt[templateIndex] = fdiffCnt
	}
}

func (p *DependencyDescriptor) frame_fdiffs() {
	FrameFdiffCnt := 0
	next_fdiff_size := p.f(2)
	for next_fdiff_size == 1 {
		fdiff_minus_one := p.f(4 * next_fdiff_size)
		p.FrameFdiff[FrameFdiffCnt] = fdiff_minus_one + 1
		FrameFdiffCnt++
		next_fdiff_size = p.f(2)
	}
}

func (p *DependencyDescriptor) template_chains() {
	p.chain_cnt = p.ns(p.DtCnt + 1)
	if p.chain_cnt == 0 {
		return
	}
	for dtIndex := 0; dtIndex < p.DtCnt; dtIndex++ {
		p.decode_target_protected_by[dtIndex] = p.ns(p.chain_cnt)
	}
	for templateIndex := 0; templateIndex < p.TemplateCnt; templateIndex++ {
		p.template_chain_fdiff[templateIndex] = make(map[int]int)
		for chainIndex := 0; chainIndex < p.chain_cnt; chainIndex++ {
			p.template_chain_fdiff[templateIndex][chainIndex] = p.f(4)
		}
	}
}

func (p *DependencyDescriptor) frame_chains() {
	for chainIndex := 0; chainIndex < p.chain_cnt; chainIndex++ {
		p.frame_chain_fdiff[chainIndex] = p.f(8)
	}
}

func (p *DependencyDescriptor) decode_target_layers() {
	for dtIndex := 0; dtIndex < p.DtCnt; dtIndex++ {
		spatialId := 0
		temporalId := 0
		for templateIndex := 0; templateIndex < p.TemplateCnt; templateIndex++ {
			if p.template_dti[templateIndex][dtIndex] != 0 {
				if p.TemplateSpatialId[templateIndex] > spatialId {
					spatialId = p.TemplateSpatialId[templateIndex]
				}
				if p.TemplateTemporalId[templateIndex] > temporalId {
					temporalId = p.TemplateTemporalId[templateIndex]
				}
			}
		}
		p.DecodeTargetSpatialId[dtIndex] = spatialId
		p.DecodeTargetTemporalId[dtIndex] = temporalId
	}
}
