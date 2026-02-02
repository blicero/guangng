// /home/krylon/go/src/github.com/blicero/guang/frontend/html/static/controlpanel.js
// -*- mode: javascript; coding: utf-8; -*-
// Time-stamp: <2026-01-31 13:22:29 krylon>
// Copyright 2022 Benjamin Walkenhorst

'use strict'

var count = {
    'GenAddr': 0,
    'GenName': 0,
    'Scanner': 0,
    'XFR':     0,
}

const cntID = {
    'GenAddr': '#cnt_gen_addr',
    'GenName': '#cnt_gen_name',
    'Scanner': '#cnt_scan',
    'XFR':     '#cnt_xfr',
}

const amtID = {
    'GenAddr':   '#amt_gen_addr',
    'GenName':   '#amt_gen_name',
    'Scanner':   '#amt_scan',
    'XFR':       '#amt_xfr',
}

function workerSpawn(fac) {
    const amt = $(amtID[fac])[0].value
    const addr = `/ajax/spawn_worker/${facilities[fac]}/${amt}`
    const req = $.get(addr,
                      {},
                      (res) => {
                          if (res.Status) {
                              const counterID = cntID[fac]
                              // Update panel?
                              $(counterID)[0].innerHTML = res.NewCnt
                          } else {
                              // alert(res.Message)
                              appendMsg(res.Message)
                          }
                      },
                      'json'
                     ).fail((reply, status, txt) => {
                         const msg = `Failed to load update: ${status} -- ${reply} -- ${txt}`
                         console.log(msg)
                         //alert(msg)
                         appendMsg(msg)
                     })
} // function spawn(fac)

function workerStop(fac) {
    const amt = $(amtID[fac])[0].value
    const addr = `/ajax/stop_worker/${facilities[fac]}/${amt}`

    const req = $.get(
        addr,
        {},
        (res) => {
            if (res.Status) {
                const counterID = cntID[fac]

                // Update panel
                $(counterID)[0].innerHTML = res.NewCnt
            } else {
                // alert(res.Message)
                appendMsg(res.Message)
            }
        },
        'json'
    ).fail((reply, status, txt) => {
        const msg = `Failed to load update: ${status} -- ${reply} -- ${txt}`
        console.log(msg)
        // alert(msg)
        appendMsg(msg)
    })
} // function stop(fac)

function loadWorkerCount() {
    const addr = '/ajax/worker_count'

    try {
        let req = $.get(
            addr,
            {},
            (res) => {
                if (res.Status) {
                    for (const [fac, id] of Object.entries(cntID)) {
                        $(id)[0].innerHTML = res[fac]
                    }
                } else {
                    const msg = `${res.Timestamp} - Error requesting worker count: ${res.Message}`
                    console.log(msg)
                    // alert(msg)
                    appendMsg(msg)
                }
            },
            'json'
        ).fail((reply, status, txt) => {
            const msg = `Failed to load worker count: ${status} -- ${reply} -- ${txt}`
            console.log(msg)
            // alert(msg)
            appendMsg(msg)
        })
    } finally {
        window.setTimeout(loadWorkerCount, 2500)
    }
} // function loadWorkerCount()
