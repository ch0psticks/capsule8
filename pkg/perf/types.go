package perf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"os"
	"sync"
	"sync/atomic"
)

const sizeofPerfEventAttr = 112

const (
	PERF_EVENT_IOC_ENABLE       uintptr = 0x2400 + 0
	PERF_EVENT_IOC_DISABLE              = 0x2400 + 1
	PERF_EVENT_IOC_REFRESH              = 0x2400 + 2
	PERF_EVENT_IOC_RESET                = 0x2400 + 3
	PERF_EVENT_IOC_PERIOD               = 0x40080000 | (0x2400 + 4)
	PERF_EVENT_IOC_SET_OUTPUT           = 0x2400 + 5
	PERF_EVENT_IOC_SET_FILTER           = 0x40080000 | (0x2400 + 6)
	PERF_EVENT_IOC_ID                   = 0x80080000 | (0x2400 + 7)
	PERF_EVENT_IOC_SET_BPF              = 0x40040000 | (0x2400 + 8)
	PERF_EVENT_IOC_PAUSE_OUTPUT         = 0x40040000 | (0x2400 + 9)
)

const (
	PERF_EVENT_IOC_FLAG_GROUP uintptr = 1 << iota
)

const (
	PERF_FLAG_FD_NO_GROUP uintptr = 1 << iota
	PERF_FLAG_FD_OUTPUT
	PERF_FLAG_PID_CGROUP
	PERF_FLAG_FD_CLOEXEC
)

const (
	PERF_FORMAT_TOTAL_TIME_ENABLED uint64 = 1 << iota
	PERF_FORMAT_TOTAL_TIME_RUNNING
	PERF_FORMAT_ID
	PERF_FORMAT_GROUP
)

const (
	PERF_RECORD_MISC_COMM_EXEC uint16 = (1 << 13)
)

const (
	PERF_TYPE_HARDWARE uint32 = iota
	PERF_TYPE_SOFTWARE
	PERF_TYPE_TRACEPOINT
	PERF_TYPE_HW_CACHE
	PERF_TYPE_RAW
	PERF_TYPE_BREAKPOINT
	PERF_TYPE_MAX
)

const (
	PERF_RECORD_INVALID uint32 = iota
	PERF_RECORD_MMAP
	PERF_RECORD_LOST
	PERF_RECORD_COMM
	PERF_RECORD_EXIT
	PERF_RECORD_THROTTLE
	PERF_RECORD_UNTHROTTLE
	PERF_RECORD_FORK
	PERF_RECORD_READ
	PERF_RECORD_SAMPLE
	PERF_RECORD_MMAP2
	PERF_RECORD_AUX
	PERF_RECORD_ITRACE_START
	PERF_RECORD_LOST_SAMPLES
	PERF_RECORD_SWITCH
	PERF_RECORD_SWITCH_CPU_WIDE
	PERF_RECORD_MAX
)

const (
	PERF_SAMPLE_IP uint64 = 1 << iota
	PERF_SAMPLE_TID
	PERF_SAMPLE_TIME
	PERF_SAMPLE_ADDR
	PERF_SAMPLE_READ
	PERF_SAMPLE_CALLCHAIN
	PERF_SAMPLE_ID
	PERF_SAMPLE_CPU
	PERF_SAMPLE_PERIOD
	PERF_SAMPLE_STREAM_ID
	PERF_SAMPLE_RAW
	PERF_SAMPLE_BRANCH_STACK
	PERF_SAMPLE_REGS_USER
	PERF_SAMPLE_STACK_USER
	PERF_SAMPLE_WEIGHT
	PERF_SAMPLE_DATA_SRC
	PERF_SAMPLE_IDENTIFIER
	PERF_SAMPLE_TRANSACTION
	PERF_SAMPLE_REGS_INTR
	PERF_SAMPLE_MAX
)

// Bitmasks for bitfield in EventAttr
const (
	eaDisabled = 1 << iota
	eaInherit
	eaPinned
	eaExclusive
	eaExclusiveUser
	eaExclusiveKernel
	eaExclusiveHV
	eaExclusiveIdle
	eaMmap
	eaComm
	eaFreq
	eaInheritStat
	eaEnableOnExec
	eaTask
	eaWatermark
	eaPreciseIP1
	eaPreciseIP2
	eaMmapData
	eaSampleIDAll
	eaExcludeHost
	eaExcludeGuest
	eaExcludeCallchainKernel
	eaExcludeCallchainUser
	eaMmap2
	eaCommExec
	eaUseClockID
	eaContextSwitch
)

/*
   struct perf_event_attr {
       __u32 type;         // Type of event
       __u32 size;         // Size of attribute structure
       __u64 config;       // Type-specific configuration

       union {
           __u64 sample_period;    // Period of sampling
           __u64 sample_freq;      // Frequency of sampling
       };

       __u64 sample_type;  // Specifies values included in sample
       __u64 read_format;  // Specifies values returned in read

       __u64 disabled       : 1,   // off by default
             inherit        : 1,   // children inherit it
             pinned         : 1,   // must always be on PMU
             exclusive      : 1,   // only group on PMU
             exclude_user   : 1,   // don't count user
             exclude_kernel : 1,   // don't count kernel
             exclude_hv     : 1,   // don't count hypervisor
             exclude_idle   : 1,   // don't count when idle
             mmap           : 1,   // include mmap data
             comm           : 1,   // include comm data
             freq           : 1,   // use freq, not period
             inherit_stat   : 1,   // per task counts
             enable_on_exec : 1,   // next exec enables
             task           : 1,   // trace fork/exit
             watermark      : 1,   // wakeup_watermark
             precise_ip     : 2,   // skid constraint
             mmap_data      : 1,   // non-exec mmap data
             sample_id_all  : 1,   // sample_type all events
             exclude_host   : 1,   // don't count in host
             exclude_guest  : 1,   // don't count in guest
             exclude_callchain_kernel : 1,
                                   // exclude kernel callchains
             exclude_callchain_user   : 1,
                                   // exclude user callchains
             mmap2          :  1,  // include mmap with inode data
             comm_exec      :  1,  // flag comm events that are due to exec
             use_clockid    :  1,  // use clockid for time fields
             context_switch :  1,  // context switch data

             __reserved_1   : 37;

       union {
           __u32 wakeup_events;    // wakeup every n events
           __u32 wakeup_watermark; // bytes before wakeup
       };

       __u32     bp_type;          // breakpoint type

       union {
           __u64 bp_addr;          // breakpoint address
           __u64 config1;          // extension of config
       };

       union {
           __u64 bp_len;           // breakpoint length
           __u64 config2;          // extension of config1
       };
       __u64 branch_sample_type;   // enum perf_branch_sample_type
       __u64 sample_regs_user;     // user regs to dump on samples
       __u32 sample_stack_user;    // size of stack to dump on samples
       __s32 clockid;              // clock to use for time fields
       __u64 sample_regs_intr;     // regs to dump on samples
       __u32 aux_watermark;        // aux bytes before wakeup
       __u16 sample_max_stack;     // max frames in callchain
       __u16 __reserved_2;         // align to u64
   };
*/

// EventAttr is a translation of the Linux kernel's struct perf_event_attr
// into Go. It provides detailed configuration information for the event
// being created.
type EventAttr struct {
	Type                   uint32
	Size                   uint32
	Config                 uint64
	SamplePeriod           uint64
	SampleFreq             uint64
	SampleType             uint64
	ReadFormat             uint64
	Disabled               bool
	Inherit                bool
	Pinned                 bool
	Exclusive              bool
	ExclusiveUser          bool
	ExclusiveKernel        bool
	ExclusiveHV            bool
	ExclusiveIdle          bool
	Mmap                   bool
	Comm                   bool
	Freq                   bool
	InheritStat            bool
	EnableOnExec           bool
	Task                   bool
	Watermark              bool
	PreciseIP              uint8
	MmapData               bool
	SampleIDAll            bool
	ExcludeHost            bool
	ExcludeGuest           bool
	ExcludeCallchainKernel bool
	ExcludeCallchainUser   bool
	Mmap2                  bool
	CommExec               bool
	UseClockID             bool
	ContextSwitch          bool
	WakeupEvents           uint32
	WakeupWatermark        uint32
	BPType                 uint32
	BPAddr                 uint64
	Config1                uint64
	BPLen                  uint64
	Config2                uint64
	BranchSampleType       uint64
	SampleRegsUser         uint64
	SampleStackUser        uint32
	ClockID                int32
	SampleRegsIntr         uint64
	AuxWatermark           uint32
	SampleMaxStack         uint16
}

// write serializes the EventAttr as a perf_event_attr struct compatible
// with the kernel.
func (ea *EventAttr) write(buf io.Writer) error {
	binary.Write(buf, binary.LittleEndian, ea.Type)
	binary.Write(buf, binary.LittleEndian, ea.Size)
	binary.Write(buf, binary.LittleEndian, ea.Config)

	if (ea.Freq && ea.SamplePeriod != 0) ||
		(!ea.Freq && ea.SampleFreq != 0) {
		return errors.New("Encoding error: invalid SamplePeriod/SampleFreq union")
	}

	if ea.Freq {
		binary.Write(buf, binary.LittleEndian, ea.SampleFreq)
	} else {
		binary.Write(buf, binary.LittleEndian, ea.SamplePeriod)
	}

	binary.Write(buf, binary.LittleEndian, ea.SampleType)
	binary.Write(buf, binary.LittleEndian, ea.ReadFormat)

	bitfield := uint64(0)
	if ea.Disabled {
		bitfield |= eaDisabled
	}
	if ea.Inherit {
		bitfield |= eaInherit
	}
	if ea.Pinned {
		bitfield |= eaPinned
	}
	if ea.Exclusive {
		bitfield |= eaExclusive
	}
	if ea.ExclusiveUser {
		bitfield |= eaExclusiveUser
	}
	if ea.ExclusiveKernel {
		bitfield |= eaExclusiveKernel
	}
	if ea.ExclusiveHV {
		bitfield |= eaExclusiveHV
	}
	if ea.Mmap {
		bitfield |= eaMmap
	}
	if ea.Comm {
		bitfield |= eaComm
	}
	if ea.Freq {
		bitfield |= eaFreq
	}
	if ea.InheritStat {
		bitfield |= eaInheritStat
	}
	if ea.EnableOnExec {
		bitfield |= eaEnableOnExec
	}
	if ea.Task {
		bitfield |= eaTask
	}
	if ea.Watermark {
		bitfield |= eaWatermark
	}

	if ea.PreciseIP > 3 {
		return errors.New("Encoding error: PreciseIP must be < 4")
	}

	if ea.PreciseIP&0x1 != 0 {
		bitfield |= eaPreciseIP1
	}
	if ea.PreciseIP&0x2 != 0 {
		bitfield |= eaPreciseIP2
	}

	if ea.MmapData {
		bitfield |= eaMmapData
	}
	if ea.SampleIDAll {
		bitfield |= eaSampleIDAll
	}
	if ea.ExcludeHost {
		bitfield |= eaExcludeHost
	}
	if ea.ExcludeGuest {
		bitfield |= eaExcludeGuest
	}
	if ea.ExcludeCallchainKernel {
		bitfield |= eaExcludeCallchainKernel
	}
	if ea.ExcludeCallchainUser {
		bitfield |= eaExcludeCallchainUser
	}
	if ea.Mmap2 {
		bitfield |= eaMmap2
	}
	if ea.CommExec {
		bitfield |= eaCommExec
	}
	if ea.UseClockID {
		bitfield |= eaUseClockID
	}
	if ea.ContextSwitch {
		bitfield |= eaContextSwitch
	}
	binary.Write(buf, binary.LittleEndian, bitfield)

	if (ea.Watermark && ea.WakeupEvents != 0) ||
		(!ea.Watermark && ea.WakeupWatermark != 0) {
		return errors.New("Encoding error: invalid WakeupWatermark/WakeupEvents union")
	}

	if ea.Watermark {
		binary.Write(buf, binary.LittleEndian, ea.WakeupWatermark)
	} else {
		binary.Write(buf, binary.LittleEndian, ea.WakeupEvents)
	}

	binary.Write(buf, binary.LittleEndian, ea.BPType)

	if ea.Config1 != 0 {
		binary.Write(buf, binary.LittleEndian, ea.Config1)
	} else {
		binary.Write(buf, binary.LittleEndian, ea.BPAddr)
	}

	if ea.Config2 != 0 {
		binary.Write(buf, binary.LittleEndian, ea.Config2)
	} else {
		binary.Write(buf, binary.LittleEndian, ea.BPLen)
	}

	binary.Write(buf, binary.LittleEndian, ea.BranchSampleType)
	binary.Write(buf, binary.LittleEndian, ea.SampleRegsUser)
	binary.Write(buf, binary.LittleEndian, ea.SampleStackUser)
	binary.Write(buf, binary.LittleEndian, ea.ClockID)
	binary.Write(buf, binary.LittleEndian, ea.SampleRegsIntr)
	binary.Write(buf, binary.LittleEndian, ea.AuxWatermark)
	binary.Write(buf, binary.LittleEndian, ea.SampleMaxStack)

	binary.Write(buf, binary.LittleEndian, uint16(0))

	return nil
}

/*
   struct perf_event_mmap_page {
       __u32 version;        // version number of this structure
       __u32 compat_version; // lowest version this is compat with
       __u32 lock;           // seqlock for synchronization
       __u32 index;          // hardware counter identifier
       __s64 offset;         // add to hardware counter value
       __u64 time_enabled;   // time event active
       __u64 time_running;   // time event on CPU
       union {
           __u64   capabilities;
           struct {
               __u64 cap_usr_time / cap_usr_rdpmc / cap_bit0 : 1,
                     cap_bit0_is_deprecated : 1,
                     cap_user_rdpmc         : 1,
                     cap_user_time          : 1,
                     cap_user_time_zero     : 1,
           };
       };
       __u16 pmc_width;
       __u16 time_shift;
       __u32 time_mult;
       __u64 time_offset;
       __u64 __reserved[120];   // Pad to 1k
       __u64 data_head;         // head in the data section
       __u64 data_tail;         // user-space written tail
       __u64 data_offset;       // where the buffer starts
       __u64 data_size;         // data buffer size
       __u64 aux_head;
       __u64 aux_tail;
       __u64 aux_offset;
       __u64 aux_size;
   }
*/

type metadata struct {
	Version       uint32
	CompatVersion uint32
	Lock          uint32
	Index         uint32
	Offset        int64
	TimeEnabled   uint64
	TimeRemaining uint64
	Capabilities  uint64
	PMCWidth      uint16
	TimeWidth     uint16
	TimeMult      uint32
	TimeOffset    uint64
	_             [120]uint64
	DataHead      uint64
	DataTail      uint64
	DataOffset    uint64
	DataSize      uint64
	AuxHead       uint64
	AuxTail       uint64
	AuxOffset     uint64
	AuxSize       uint64
}

// -----------------------------------------------------------------------------

/*
   struct perf_event_header {
       __u32   type;
       __u16   misc;
       __u16   size;
   };
*/

type eventHeader struct {
	Type uint32
	Misc uint16
	Size uint16
}

func (eh *eventHeader) read(reader *bytes.Reader) error {
	err := binary.Read(reader, binary.LittleEndian, eh)
	if err != nil {
		return err
	}

	return nil
}

type CommRecord struct {
	Pid  uint32
	Tid  uint32
	Comm []byte
}

func (cr *CommRecord) read(reader *bytes.Reader) error {
	err := binary.Read(reader, binary.LittleEndian, &cr.Pid)
	if err != nil {
		return err
	}

	err = binary.Read(reader, binary.LittleEndian, &cr.Tid)
	if err != nil {
		return err
	}

	cr.Comm = make([]byte, 0)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return err
		}
		if b != 0 {
			cr.Comm = append(cr.Comm, b)
		} else {
			break
		}
	}

	// The comm[] field is NULL-padded up to an 8-byte aligned
	// offset. Discard the rest of the bytes.
	for i := len(cr.Comm) + 1; (i % 8) != 0; i++ {
		reader.ReadByte()
	}

	return nil
}

type ExitRecord struct {
	Pid  uint32
	Ppid uint32
	Tid  uint32
	Ptid uint32
	Time uint64
}

func (er *ExitRecord) read(reader *bytes.Reader) error {
	err := binary.Read(reader, binary.LittleEndian, er)
	if err != nil {
		return err
	}

	return nil
}

type ForkRecord struct {
	Pid  uint32
	Ppid uint32
	Tid  uint32
	Ptid uint32
	Time uint64
}

func (fr *ForkRecord) read(reader *bytes.Reader) error {
	err := binary.Read(reader, binary.LittleEndian, fr)
	if err != nil {
		return err
	}

	return nil
}

// CounterValue resepresents the read value of a counter event
type CounterValue struct {
	// Globally unique identifier for this counter event. Only
	// present if PERF_FORMAT_ID was specified.
	ID uint64

	// The counter result
	Value uint64
}

// CounterGroup represents the read values of a group of counter events
type CounterGroup struct {
	TimeEnabled uint64
	TimeRunning uint64
	Values      []CounterValue
}

func (cg *CounterGroup) read(reader *bytes.Reader, format uint64) error {
	var err error

	if (format & PERF_FORMAT_GROUP) != 0 {
		var nr uint64
		err = binary.Read(reader, binary.LittleEndian, &nr)
		if err != nil {
			return err
		}

		if (format & PERF_FORMAT_TOTAL_TIME_ENABLED) != 0 {
			err = binary.Read(reader, binary.LittleEndian,
				&cg.TimeEnabled)
			if err != nil {
				return err
			}
		}

		if (format & PERF_FORMAT_TOTAL_TIME_RUNNING) != 0 {
			err = binary.Read(reader, binary.LittleEndian,
				&cg.TimeRunning)
			if err != nil {
				return err
			}
		}

		var values []CounterValue

		for i := uint64(0); i < nr; i++ {
			value := CounterValue{}
			err = binary.Read(reader, binary.LittleEndian,
				&value.Value)
			if err != nil {
				return err
			}

			if (format & PERF_FORMAT_ID) != 0 {
				err = binary.Read(reader, binary.LittleEndian,
					&value.ID)
				if err != nil {
					return err
				}
			}

			values = append(values, value)
		}

		return nil
	}

	value := CounterValue{}

	err = binary.Read(reader, binary.LittleEndian, &value.Value)
	if err != nil {
		return err
	}

	if (format & PERF_FORMAT_TOTAL_TIME_ENABLED) != 0 {
		err = binary.Read(reader, binary.LittleEndian,
			&cg.TimeEnabled)
		if err != nil {
			return err
		}
	}

	if (format & PERF_FORMAT_TOTAL_TIME_RUNNING) != 0 {
		err = binary.Read(reader, binary.LittleEndian,
			&cg.TimeRunning)
		if err != nil {
			return err
		}
	}

	if (format & PERF_FORMAT_ID) != 0 {
		err = binary.Read(reader, binary.LittleEndian, &value.ID)
		if err != nil {
			return err
		}
	}

	cg.Values = append(cg.Values, value)
	return nil
}

type BranchEntry struct {
	From      uint64
	To        uint64
	Mispred   bool
	Predicted bool
	InTx      bool
	Abort     bool
	Cycles    uint16
}

type SampleRecord struct {
	SampleID    uint64
	IP          uint64
	Pid         uint32
	Tid         uint32
	Time        uint64
	Addr        uint64
	ID          uint64
	StreamID    uint64
	CPU         uint32
	Period      uint64
	V           CounterGroup
	IPs         []uint64
	RawData     []byte
	Branches    []BranchEntry
	UserABI     uint64
	UserRegs    []uint64
	StackData   []uint64
	Weight      uint64
	DataSrc     uint64
	Transaction uint64
	IntrABI     uint64
	IntrRegs    []uint64
}

func (s *SampleRecord) read(reader *bytes.Reader, sampleType uint64, readFormat uint64) error {
	if (sampleType & PERF_SAMPLE_IDENTIFIER) != 0 {
		binary.Read(reader, binary.LittleEndian, &s.SampleID)
	}

	if (sampleType & PERF_SAMPLE_IP) != 0 {
		binary.Read(reader, binary.LittleEndian, &s.IP)
	}

	if (sampleType & PERF_SAMPLE_TID) != 0 {
		binary.Read(reader, binary.LittleEndian, &s.Pid)
		binary.Read(reader, binary.LittleEndian, &s.Tid)
	}

	if (sampleType & PERF_SAMPLE_TIME) != 0 {
		binary.Read(reader, binary.LittleEndian, &s.Time)
	}

	if (sampleType & PERF_SAMPLE_ADDR) != 0 {
		binary.Read(reader, binary.LittleEndian, &s.Addr)
	}

	if (sampleType & PERF_SAMPLE_ID) != 0 {
		binary.Read(reader, binary.LittleEndian, &s.ID)
	}

	if (sampleType & PERF_SAMPLE_STREAM_ID) != 0 {
		binary.Read(reader, binary.LittleEndian, &s.StreamID)
	}

	if (sampleType & PERF_SAMPLE_CPU) != 0 {
		res := uint32(0)

		binary.Read(reader, binary.LittleEndian, &s.CPU)
		binary.Read(reader, binary.LittleEndian, &res)
	}

	if (sampleType & PERF_SAMPLE_PERIOD) != 0 {
		binary.Read(reader, binary.LittleEndian, &s.Period)
	}

	if (sampleType & PERF_SAMPLE_READ) != 0 {
		s.V.read(reader, readFormat)
	}

	if (sampleType & PERF_SAMPLE_CALLCHAIN) != 0 {
		var nr uint64

		binary.Read(reader, binary.LittleEndian, &nr)

		for i := uint64(0); i < nr; i++ {
			var ip uint64
			binary.Read(reader, binary.LittleEndian, &ip)
			s.IPs = append(s.IPs, ip)
		}

	}

	if (sampleType & PERF_SAMPLE_RAW) != 0 {
		rawDataSize := uint32(0)
		binary.Read(reader, binary.LittleEndian, &rawDataSize)
		s.RawData = make([]byte, rawDataSize)
		binary.Read(reader, binary.LittleEndian, s.RawData)
	}

	if (sampleType & PERF_SAMPLE_BRANCH_STACK) != 0 {
		bnr := uint64(0)
		binary.Read(reader, binary.LittleEndian, &bnr)

		for i := uint64(0); i < bnr; i++ {
			var pbe struct {
				from  uint64
				to    uint64
				flags uint64
			}
			binary.Read(reader, binary.LittleEndian, &pbe)

			branchEntry := BranchEntry{
				From:      pbe.from,
				To:        pbe.to,
				Mispred:   (pbe.flags&(1<<0) != 0),
				Predicted: (pbe.flags&(1<<1) != 0),
				InTx:      (pbe.flags&(1<<3) != 0),
				Abort:     (pbe.flags&(1<<4) != 0),
				Cycles:    uint16((pbe.flags & 0xff0) >> 4),
			}

			s.Branches = append(s.Branches, branchEntry)
		}
	}

	if (sampleType&PERF_SAMPLE_REGS_USER) != 0 ||
		(sampleType&PERF_SAMPLE_STACK_USER) != 0 ||
		(sampleType&PERF_SAMPLE_WEIGHT) != 0 ||
		(sampleType&PERF_SAMPLE_DATA_SRC) != 0 ||
		(sampleType&PERF_SAMPLE_TRANSACTION) != 0 ||
		(sampleType&PERF_SAMPLE_REGS_INTR) != 0 {

		panic("PERF_RECORD_SAMPLE field parsing not implemented")
	}

	return nil
}

/*
   struct sample_id {
       { u32 pid, tid; } // if PERF_SAMPLE_TID set
       { u64 time;     } // if PERF_SAMPLE_TIME set
       { u64 id;       } // if PERF_SAMPLE_ID set
       { u64 stream_id;} // if PERF_SAMPLE_STREAM_ID set
       { u32 cpu, res; } // if PERF_SAMPLE_CPU set
       { u64 id;       } // if PERF_SAMPLE_IDENTIFIER set
   };
*/

type SampleID struct {
	PID      uint32
	TID      uint32
	Time     uint64
	ID       uint64
	StreamID uint64
	CPU      uint32
}

var sampleIDSizeCache atomic.Value
var sampleIDSizeCacheMutex sync.Mutex

func (sid *SampleID) getSize(eventSampleType uint64) int {
	var cache map[uint64]int

	sizeCache := sampleIDSizeCache.Load()
	if sizeCache != nil {
		cache = sizeCache.(map[uint64]int)
		size := cache[eventSampleType]
		if size > 0 {
			return size
		}
	}

	sampleIDSizeCacheMutex.Lock()
	sizeCache = sampleIDSizeCache.Load()

	if sizeCache == nil {
		cache = make(map[uint64]int)
	} else {
		cache = sizeCache.(map[uint64]int)
	}

	size := int(0)

	if (eventSampleType & PERF_SAMPLE_TID) != 0 {
		size += binary.Size(sid.PID)
		size += binary.Size(sid.TID)
	}

	if (eventSampleType & PERF_SAMPLE_TIME) != 0 {
		size += binary.Size(sid.Time)
	}

	if (eventSampleType & PERF_SAMPLE_ID) != 0 {
		size += binary.Size(sid.ID)
	}

	if (eventSampleType & PERF_SAMPLE_STREAM_ID) != 0 {
		size += binary.Size(sid.StreamID)
	}

	if (eventSampleType & PERF_SAMPLE_CPU) != 0 {
		res := uint32(0)

		size += binary.Size(sid.CPU)
		size += binary.Size(res)
	}

	if (eventSampleType & PERF_SAMPLE_IDENTIFIER) != 0 {
		size += binary.Size(sid.ID)
	}

	cache[eventSampleType] = size
	sampleIDSizeCache.Store(cache)
	sampleIDSizeCacheMutex.Unlock()

	return size
}

func (sid *SampleID) read(reader *bytes.Reader, eventSampleType uint64) error {
	var err error

	if (eventSampleType & PERF_SAMPLE_TID) != 0 {
		err = binary.Read(reader, binary.LittleEndian, &sid.PID)
		if err != nil {
			return err
		}
		err = binary.Read(reader, binary.LittleEndian, &sid.TID)
		if err != nil {
			return err
		}
	}

	if (eventSampleType & PERF_SAMPLE_TIME) != 0 {
		err = binary.Read(reader, binary.LittleEndian, &sid.Time)
		if err != nil {
			return err
		}
	}

	if (eventSampleType & PERF_SAMPLE_ID) != 0 {
		err = binary.Read(reader, binary.LittleEndian, &sid.ID)
		if err != nil {
			return err
		}
	}

	if (eventSampleType & PERF_SAMPLE_STREAM_ID) != 0 {
		err = binary.Read(reader, binary.LittleEndian, &sid.StreamID)
		if err != nil {
			return err
		}
	}

	if (eventSampleType & PERF_SAMPLE_CPU) != 0 {
		res := uint32(0)

		err = binary.Read(reader, binary.LittleEndian, &sid.CPU)
		if err != nil {
			return err
		}
		err = binary.Read(reader, binary.LittleEndian, &res)
		if err != nil {
			return err
		}
	}

	if (eventSampleType & PERF_SAMPLE_IDENTIFIER) != 0 {
		err = binary.Read(reader, binary.LittleEndian, &sid.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

type Sample struct {
	eventHeader
	Record interface{}
	SampleID
}

func (sample *Sample) read(reader *bytes.Reader, sampleType, readFormat uint64) error {
	startPos, _ := reader.Seek(0, os.SEEK_CUR)

	err := sample.eventHeader.read(reader)
	if err != nil {
		// This is not recoverable
		panic("Could not read perf event header")
	}

	switch sample.Type {
	case PERF_RECORD_FORK:
		record := new(ForkRecord)
		err = record.read(reader)
		if err != nil {
			break
		}

		sample.Record = record

		sample.SampleID.read(reader, sampleType)

	case PERF_RECORD_EXIT:
		record := new(ExitRecord)
		err = record.read(reader)
		if err != nil {
			break
		}

		sample.Record = record

		sample.SampleID.read(reader, sampleType)

	case PERF_RECORD_COMM:
		record := new(CommRecord)
		err = record.read(reader)
		if err != nil {
			break
		}

		sample.Record = record
		sample.SampleID.read(reader, sampleType)

	case PERF_RECORD_SAMPLE:
		record := new(SampleRecord)
		err = record.read(reader, sampleType, readFormat)
		if err != nil {
			break
		}

		sample.Record = record

		// NB: PERF_RECORD_SAMPLE does not include a trailing
		// SampleID, even if SampleIDAll is true.

	default:
		//
		// Unknown type, read sample record as a byte slice
		//
		sampleIDSize := sample.SampleID.getSize(sampleType)

		recordSize := sample.Size -
			uint16(binary.Size(sample.eventHeader)) -
			uint16(sampleIDSize)

		recordData := make([]byte, recordSize)

		var n int
		n, err = reader.Read(recordData)
		if err != nil {
			break
		} else if n < int(recordSize) {
			log.Printf("Short read: %d < %d", n, int(recordSize))
			err = errors.New("Read error")
			break
		}

		sample.Record = recordData

		sample.SampleID.read(reader, sampleType)
	}

	// If there was a read error, seek to end of the record
	if err != nil {
		reader.Seek((startPos + int64(sample.Size)), os.SEEK_SET)
	}

	return err
}