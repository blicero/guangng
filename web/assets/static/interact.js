// Time-stamp: <2026-01-30 14:54:20 krylon>
// -*- mode: javascript; coding: utf-8; -*-
// Copyright 2015-2020 Benjamin Walkenhorst <krylon@gmx.net>
//
// This file has grown quite a bit larger than I had anticipated.
// It is not a /big/ problem right now, but in the long run, I will have to
// break this thing up into several smaller files.

'use strict'

function defined (x) {
    return undefined !== x && null !== x
}

function fmtDateNumber (n) {
    return (n < 10 ? '0' : '') + n.toString()
} // function fmtDateNumber(n)

function timeStampString (t) {
    if ((typeof t) === 'string') {
        return t
    }

    const year = t.getYear() + 1900
    const month = fmtDateNumber(t.getMonth() + 1)
    const day = fmtDateNumber(t.getDate())
    const hour = fmtDateNumber(t.getHours())
    const minute = fmtDateNumber(t.getMinutes())
    const second = fmtDateNumber(t.getSeconds())

    const s =
          year + '-' + month + '-' + day +
          ' ' + hour + ':' + minute + ':' + second
    return s
} // function timeStampString(t)

function fmtDuration (seconds) {
    let minutes = 0
    let hours = 0

    while (seconds > 3599) {
        hours++
        seconds -= 3600
    }

    while (seconds > 59) {
        minutes++
        seconds -= 60
    }

    if (hours > 0) {
        return `${hours}h${minutes}m${seconds}s`
    } else if (minutes > 0) {
        return `${minutes}m${seconds}s`
    } else {
        return `${seconds}s`
    }
} // function fmtDuration(seconds)

function beaconLoop () {
    try {
        if (settings.beacon.active) {
            const req = $.get('/ajax/beacon',
                              {},
                              function (response) {
                                  let status = ''

                                  if (response.Status) {
                                      status = 
                                          response.Message +
                                          ' running on ' +
                                          response.Hostname +
                                          ' is alive at ' +
                                          response.Timestamp
                                  } else {
                                      status = 'Server is not responding'
                                  }

                                  const beaconDiv = $('#beacon')[0]

                                  if (defined(beaconDiv)) {
                                      beaconDiv.innerHTML = status
                                      beaconDiv.classList.remove('error')
                                  } else {
                                      console.log('Beacon field was not found')
                                  }
                              },
                              'json'
                             ).fail(function () {
                                 const beaconDiv = $('#beacon')[0]
                                 beaconDiv.innerHTML = 'Server is not responding'
                                 beaconDiv.classList.add('error')
                                 // logMsg("ERROR", "Server is not responding");
                             })
        }
    } finally {
        window.setTimeout(beaconLoop, settings.beacon.interval)
    }
} // function beaconLoop()

function beaconToggle () {
    settings.beacon.active = !settings.beacon.active
    saveSetting('beacon', 'active', settings.beacon.active)

    if (!settings.beacon.active) {
        const beaconDiv = $('#beacon')[0]
        beaconDiv.innerHTML = 'Beacon is suspended'
        beaconDiv.classList.remove('error')
    }
} // function beaconToggle()

/*
  The ‘content’ attribute of Window objects is deprecated.  Please use ‘window.top’ instead. interact.js:125:8
  Ignoring get or set of property that has [LenientThis] because the “this” object is incorrect. interact.js:125:8

*/

function db_maintenance () {
    const maintURL = '/ajax/db_maint'

    const req = $.get(
        maintURL,
        {},
        function (res) {
            if (!res.Status) {
                console.log(res.Message)
                postMessage(new Date(), 'ERROR', res.Message)
            } else {
                const msg = 'Database Maintenance performed without errors'
                console.log(msg)
                postMessage(new Date(), 'INFO', msg)
            }
        },
        'json'
    ).fail(function () {
        const msg = 'Error performing DB maintenance'
        console.log(msg)
        postMessage(new Date(), 'ERROR', msg)
    })
} // function db_maintenance()

function msgCheckSum (timestamp, level, msg) {
    const line = [timeStampString(timestamp), level, msg].join('##')

    const cksum = sha512(line)
    return cksum
}

let curMessageCnt = 0

function post_test_msg () {
    const user = $('#msgTestText')[0]
    const msg = user.value
    const now = new Date()

    postMessage(now, 'DEBUG', msg)
} // function post_tst_msg()

function postMessage (timestamp, level, msg) {
    const row = '<tr id="msg_' +
          msgCheckSum(timestamp, level, msg) +
          '"><td>' +
          timeStampString(timestamp) +
          '</td><td>' +
          level +
          '</td><td>' +
          msg +
          '</td></tr>\n'

    msgRowAdd(row)
} // function postMessage(timestamp, level, msg)

function adjustMsgMaxCnt () {
    const cntField = $('#max_msg_cnt')[0]
    const newMax = cntField.valueAsNumber

    if (newMax < curMessageCnt) {
        const rows = $('#msg_body')[0].children

        while (rows.length > newMax) {
            rows[rows.length - 1].remove()
            curMessageCnt--
        }
    }

    saveSetting('messages', 'maxShow', newMax)
} // function adjustMaxMsgCnt()

function adjustMsgCheckInterval () {
    const intervalField = $('#msg_check_interval')[0]
    if (intervalField.checkValidity()) {
        const interval = intervalField.valueAsNumber
        // intervalField.setInterval(interval); // ???
        saveSetting('messages', 'interval', interval)
    }
} // function adjustMsgCheckInterval()

function toggleCheckMessages () {
    const box = $('#msg_check_switch')[0]
    const newVal = box.checked

    saveSetting('messages', 'queryEnabled', newVal)
} // function toggleCheckMessages()

function getNewMessages () {
    // const msgURL = '/ajax/get_messages'

    // try {
    //     if (!settings.messages.queryEnabled) {
    //         return
    //     }

    //     const req = $.get(
    //         msgURL,
    //         {},
    //         function (res) {
    //             if (!res.Status) {
    //                 const msg = msgURL +
    //                       ' failed: ' +
    //                       res.Message

    //                 console.log(msg)
    //                 alert(msg)
    //             } else {
    //                 let i = 0
    //                 for (i = 0; i < res.Messages.length; i++) {
    //                     const item = res.Messages[i]
    //                     const rowid =
    //                           'msg_' +
    //                           msgCheckSum(item.Time, item.Level, item.Message)
    //                     const row = '<tr id="' +
    //                           rowid +
    //                           '"><td>' +
    //                           item.Time +
    //                           '</td><td>' +
    //                           item.Level +
    //                           '</td><td>' +
    //                           item.Message +
    //                           '</td><td>' +
    //                           '<input type="button" value="Delete" onclick="msgRowDelete(\'' +
    //                           rowid +
    //                           '\');" />' +
    //                           '</td></tr>\n'

    //                     msgRowAdd(row)
    //                 }
    //             }
    //         },
    //         'json'
    //     )
    // } finally {
    //     window.setTimeout(getNewMessages, settings.messages.interval)
    // }
} // function getNewMessages()

function logMsg (level, msg) {
    const timestamp = timeStampString(new Date())
    const rowID = 'msg_' + sha512(msgCheckSum(timestamp, level, msg))
    const row = '<tr id="' +
          rowID +
          '"><td>' +
          timestamp +
          '</td><td>' +
          level +
          '</td><td>' +
          msg +
          '</td><td>' +
          '<input type="button" value="Delete" onclick="msgRowDelete(\'' +
          rowID +
          '\');" />' +
          '</td></tr>\n'

    $('#msg_display_tbl')[0].innerHTML += row
} // function logMsg(level, msg)

function msgRowAdd (row) {
    const msgBody = $('#msg_body')[0]

    msgBody.innerHTML = row + msgBody.innerHTML

    if (++curMessageCnt > settings.messages.maxShow) {
        msgBody.children[msgBody.children.length - 1].remove()
    }

    const tbl = $('#msg_tbl')[0]
    if (tbl.hidden) {
        tbl.hidden = false
    }
} // function msgRowAdd(row)

function msgRowDelete (rowID) {
    const row = $('#' + rowID)[0]

    if (row != undefined) {
        row.remove()
        if (--curMessageCnt == 0) {
            const tbl = $('#msg_tbl')[0]
            tbl.hidden = true
        }
    }
} // function msgRowDelete(rowID)

function msgRowDeleteAll () {
    const msgBody = $('#msg_body')[0]
    msgBody.innerHTML = ''
    curMessageCnt = 0

    const tbl = $('#msg_tbl')[0]
    tbl.hidden = true
} // function msgRowDeleteAll()

function requestTestMessages () {
    const urlRoot = '/ajax/rnd_message/'

    const cnt = $('#msg_cnt')[0].valueAsNumber
    const rounds = $('#msg_round_cnt')[0].valueAsNumber
    const delay = $('#msg_round_delay')[0].valueAsNumber

    if (cnt == 0) {
        console.log('Generate *0* messages? Alrighty then...')
        return
    }

    const reqURL = urlRoot + cnt

    $.get(
        reqURL,
        {
            Rounds: rounds,
            Delay: delay
        },
        (res) => {
            if (!res.Status) {
                console.log(res.Message)
                alert(res.Message)
            }
        },
        'json'
    ).fail(function () {
        const msg = 'Requesting test messages failed.'
        console.log(msg)
        // alert(msg);
        logMsg('ERROR', msg)
    })
} // function requestTestMessages()

function toggleMsgTestDisplayVisible () {
    const tbl = $('#test_msg_cfg')[0]

    if (tbl.hidden) {
        tbl.hidden = false

        const checkbox = $('#msg_check_switch')[0]
        settings.messages.queryEnabled = checkbox.checked
    } else {
        settings.messages.queryEnabled = false
        tbl.hidden = true
    }
} // function toggleMsgTmpDisplayVisible()

function toggleMsgDisplayVisible () {
    const display = $('#msg_display_div')[0]

    display.hidden = !display.hidden
} // function toggleMsgDisplayVisible()

// Found here: https://stackoverflow.com/questions/3971841/how-to-resize-images-proportionally-keeping-the-aspect-ratio#14731922
function shrink_img (srcWidth, srcHeight, maxWidth, maxHeight) {
    const ratio = Math.min(maxWidth / srcWidth, maxHeight / srcHeight)

    return { width: srcWidth * ratio, height: srcHeight * ratio }
} // function shrink_img(srcWidth, srcHeight, maxWidth, maxHeight)

function shrink_images () {
    const selector = 'table.items img'
    const maxHeight = 300
    const maxWidth = 300

    $(selector).each(function () {
        const img = $(this)[0]
        if (img.width > maxWidth || img.height > maxHeight) {
            const size = shrink_img(img.width, img.height, maxWidth, maxHeight)

            img.width = size.width
            img.height = size.height
        }
    })
} // function shrink_images()

function shutdown_server () {
    const url = '/ajax/shutdown'

    if (!confirm('Shut down server?')) {
        return false
    }

    const req = $.get(url,
                      { AreYouSure: true, AreYouReallySure: true },
                      function (reply) {
                          if (!reply.Status) {
                              const msg = `Error shutting down Server: ${reply.Message}`
                              console.log(msg)
                              alert(msg)
                          }
                      },
                      'json')

    req.fail(function (reply, status_text, xhr) {
        const msg = `Error getting Items: ${status_text} - ${xhr}`
        console.log(msg)
        alert(msg)
    })
} // function shutdown_server()

function page_frame_resize () {
    const h = window.innerHeight
    const w = window.innerWidth
    const frameID = "#page_frame"
    const frame = $(frameID)[0]

    const msg = `Window size is ${w}x${h}`

    console.log(msg)
    alert(msg)
} // function page_frame_resize ()
