//go:build linux

package inputshare

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

// recordConn is a dedicated, hand-rolled X11 connection used for exactly one
// request: RECORD's EnableContext.
//
// It exists because that request breaks every general-purpose X client
// library's request/reply bookkeeping: the server keeps streaming further
// reply packets under the original request's sequence number for as long as
// the context stays enabled. xgb matches each incoming reply against a cookie
// that it *consumes* from its cookie queue (see readResponses in xgb.go), so
// the second RECORD chunk finds no cookie for its sequence number, drains and
// discards the cookies of unrelated in-flight requests looking for one, and
// from then on the connection's sequence accounting is broken: capture stops
// after the first chunk and any later checked request on the same connection
// blocks forever waiting for a reply that was handed to the wrong cookie.
//
// This mirrors what libXtst does in C — XRecordQueryVersion et al. on a
// "control" display, XRecordEnableContext on a second "data" display — with
// the control half staying on the ordinary xgb connection (see
// capture_linux.go) and only the streaming half living here.
//
// The connection setup and .Xauthority parsing below are ported from
// BurntSushi/xgb's conn.go and auth.go (BSD 3-clause), since that package
// keeps them unexported.
type recordConn struct {
	conn        net.Conn
	majorOpcode byte
	seq         uint16
}

// recordChunk is one EnableContext reply: a category plus the raw core
// protocol event bytes it carries.
type recordChunk struct {
	Category byte
	Data     []byte
}

const (
	xQueryExtensionOpcode = 98
	xRecordEnableContext  = 5
)

// dialRecordConn opens a second connection to the X server named by DISPLAY,
// completes the setup handshake, and resolves RECORD's major opcode on it.
func dialRecordConn() (*recordConn, error) {
	host, display, netConn, err := dialX(os.Getenv("DISPLAY"))
	if err != nil {
		return nil, err
	}

	r := &recordConn{conn: netConn}
	if err := r.handshake(host, display); err != nil {
		netConn.Close()
		return nil, err
	}

	opcode, err := r.queryExtension("RECORD")
	if err != nil {
		netConn.Close()
		return nil, err
	}
	r.majorOpcode = opcode

	return r, nil
}

// dialX parses a DISPLAY string and connects to the server it names. It
// returns the host and display number the caller needs for authority lookup.
func dialX(display string) (host, displayNum string, conn net.Conn, err error) {
	original := display
	if display == "" {
		return "", "", nil, errors.New("inputshare/x11: DISPLAY is not set")
	}

	colon := strings.LastIndex(display, ":")
	if colon < 0 {
		return "", "", nil, fmt.Errorf("inputshare/x11: bad DISPLAY %q", original)
	}

	var protocol, socket string
	if display[0] == '/' {
		socket = display[:colon]
	} else if slash := strings.LastIndex(display[:colon], "/"); slash >= 0 {
		protocol = display[:slash]
		host = display[slash+1 : colon]
	} else {
		host = display[:colon]
	}

	rest := display[colon+1:]
	if rest == "" {
		return "", "", nil, fmt.Errorf("inputshare/x11: bad DISPLAY %q", original)
	}
	displayNum = rest
	if dot := strings.LastIndex(rest, "."); dot >= 0 {
		displayNum = rest[:dot]
	}
	num, convErr := strconv.Atoi(displayNum)
	if convErr != nil || num < 0 {
		return "", "", nil, fmt.Errorf("inputshare/x11: bad DISPLAY %q", original)
	}

	switch {
	case socket != "":
		conn, err = net.Dial("unix", socket+":"+displayNum)
	case host != "" && host != "unix":
		if protocol == "" {
			protocol = "tcp"
		}
		conn, err = net.Dial(protocol, host+":"+strconv.Itoa(6000+num))
	default:
		host = ""
		conn, err = net.Dial("unix", "/tmp/.X11-unix/X"+displayNum)
	}
	if err != nil {
		return "", "", nil, fmt.Errorf("inputshare/x11: cannot connect to %s: %w", original, err)
	}

	return host, displayNum, conn, nil
}

// handshake performs the X11 connection setup, authenticating with the
// MIT-MAGIC-COOKIE-1 entry from the authority file when one is available.
func (r *recordConn) handshake(host, display string) error {
	authName, authData, err := readXAuthority(host, display)
	if err != nil {
		// Same fallback as xgb: a server with no access control still
		// accepts an unauthenticated connection.
		authName, authData = "", nil
	} else if authName != "MIT-MAGIC-COOKIE-1" || len(authData) != 16 {
		return fmt.Errorf("inputshare/x11: unsupported auth protocol %q", authName)
	}

	req := make([]byte, 12+pad4(len(authName))+pad4(len(authData)))
	req[0] = 0x6c // little endian
	binary.LittleEndian.PutUint16(req[2:], 11)
	binary.LittleEndian.PutUint16(req[4:], 0)
	binary.LittleEndian.PutUint16(req[6:], uint16(len(authName)))
	binary.LittleEndian.PutUint16(req[8:], uint16(len(authData)))
	copy(req[12:], authName)
	copy(req[12+pad4(len(authName)):], authData)

	if _, err := r.conn.Write(req); err != nil {
		return fmt.Errorf("inputshare/x11: setup write failed: %w", err)
	}

	head := make([]byte, 8)
	if _, err := io.ReadFull(r.conn, head); err != nil {
		return fmt.Errorf("inputshare/x11: setup read failed: %w", err)
	}
	code := head[0]
	reasonLen := int(head[1])
	dataLen := int(binary.LittleEndian.Uint16(head[6:]))

	body := make([]byte, dataLen*4)
	if _, err := io.ReadFull(r.conn, body); err != nil {
		return fmt.Errorf("inputshare/x11: setup read failed: %w", err)
	}
	if code == 0 {
		if reasonLen > len(body) {
			reasonLen = len(body)
		}
		return fmt.Errorf("inputshare/x11: connection refused by X server: %s", body[:reasonLen])
	}

	return nil
}

// queryExtension resolves an extension's major opcode on this connection.
func (r *recordConn) queryExtension(name string) (byte, error) {
	n := len(name)
	req := make([]byte, 8+pad4(n))
	req[0] = xQueryExtensionOpcode
	binary.LittleEndian.PutUint16(req[2:], uint16(len(req)/4))
	binary.LittleEndian.PutUint16(req[4:], uint16(n))
	copy(req[8:], name)

	r.seq++
	if _, err := r.conn.Write(req); err != nil {
		return 0, fmt.Errorf("inputshare/x11: QueryExtension write failed: %w", err)
	}

	reply := make([]byte, 32)
	if _, err := io.ReadFull(r.conn, reply); err != nil {
		return 0, fmt.Errorf("inputshare/x11: QueryExtension read failed: %w", err)
	}
	if reply[0] == 0 {
		return 0, fmt.Errorf("inputshare/x11: QueryExtension(%s) returned X error code %d", name, reply[1])
	}
	if reply[8] == 0 {
		return 0, fmt.Errorf("inputshare/x11: %s extension not present on this display", name)
	}

	return reply[9], nil
}

// enableContext issues RecordEnableContext for an already-created context.
// It does not wait for a reply: every reply is a data chunk, delivered by
// successive calls to next.
func (r *recordConn) enableContext(context uint32) error {
	req := make([]byte, 8)
	req[0] = r.majorOpcode
	req[1] = xRecordEnableContext
	binary.LittleEndian.PutUint16(req[2:], uint16(len(req)/4))
	binary.LittleEndian.PutUint32(req[4:], context)

	r.seq++
	if _, err := r.conn.Write(req); err != nil {
		return fmt.Errorf("inputshare/x11: RECORD EnableContext write failed: %w", err)
	}

	return nil
}

// next blocks until the server sends the next EnableContext chunk. Events are
// skipped (this connection never selects for any) and an X error terminates
// the stream, which is how a DisableContext or a closed connection surfaces.
func (r *recordConn) next() (recordChunk, error) {
	header := make([]byte, 32)
	for {
		if _, err := io.ReadFull(r.conn, header); err != nil {
			return recordChunk{}, err
		}

		switch header[0] {
		case 0: // X error
			return recordChunk{}, fmt.Errorf("inputshare/x11: RECORD stream returned X error code %d", header[1])

		case 1: // reply — an EnableContext data chunk
			// Length counts 4-byte units of payload beyond the 32-byte header.
			extra := int(binary.LittleEndian.Uint32(header[4:])) * 4
			chunk := recordChunk{Category: header[1]}
			if extra > 0 {
				chunk.Data = make([]byte, extra)
				if _, err := io.ReadFull(r.conn, chunk.Data); err != nil {
					return recordChunk{}, err
				}
			}
			return chunk, nil

		default: // event — nothing on this connection asked for one
			continue
		}
	}
}

func (r *recordConn) Close() error {
	return r.conn.Close()
}

func pad4(n int) int { return (n + 3) &^ 3 }

// readXAuthority returns the auth entry matching the given host and display,
// ported from xgb's auth.go.
func readXAuthority(hostname, display string) (name string, data []byte, err error) {
	const (
		familyLocal = 256
		familyWild  = 65535
	)

	if hostname == "" || hostname == "localhost" {
		hostname, err = os.Hostname()
		if err != nil {
			return "", nil, err
		}
	}

	fname := os.Getenv("XAUTHORITY")
	if fname == "" {
		home := os.Getenv("HOME")
		if home == "" {
			return "", nil, errors.New("inputshare/x11: neither $XAUTHORITY nor $HOME is set")
		}
		fname = home + "/.Xauthority"
	}

	f, err := os.Open(fname)
	if err != nil {
		return "", nil, err
	}
	defer f.Close()

	buf := make([]byte, 256)
	for {
		var family uint16
		if err := binary.Read(f, binary.BigEndian, &family); err != nil {
			return "", nil, err
		}
		addr, err := xauthString(f, buf)
		if err != nil {
			return "", nil, err
		}
		disp, err := xauthString(f, buf)
		if err != nil {
			return "", nil, err
		}
		entryName, err := xauthString(f, buf)
		if err != nil {
			return "", nil, err
		}
		entryData, err := xauthBytes(f, buf)
		if err != nil {
			return "", nil, err
		}

		addrMatch := family == familyWild || (family == familyLocal && addr == hostname)
		dispMatch := disp == "" || disp == display
		if addrMatch && dispMatch {
			// entryData aliases buf, which the next iteration would reuse.
			out := make([]byte, len(entryData))
			copy(out, entryData)
			return entryName, out, nil
		}
	}
}

func xauthBytes(r io.Reader, buf []byte) ([]byte, error) {
	var n uint16
	if err := binary.Read(r, binary.BigEndian, &n); err != nil {
		return nil, err
	}
	if int(n) > len(buf) {
		return nil, errors.New("inputshare/x11: Xauthority field too long")
	}
	if _, err := io.ReadFull(r, buf[:n]); err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func xauthString(r io.Reader, buf []byte) (string, error) {
	b, err := xauthBytes(r, buf)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
