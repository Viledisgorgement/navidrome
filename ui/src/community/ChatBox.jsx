import React, { useState, useEffect, useCallback, useRef, useMemo } from 'react'
import { useSelector } from 'react-redux'
import { useTranslate, useNotify } from 'react-admin'
import {
  Card,
  CardContent,
  Typography,
  TextField,
  IconButton,
  Button,
  Tooltip,
  makeStyles,
} from '@material-ui/core'
import SendIcon from '@material-ui/icons/Send'
import PhotoIcon from '@material-ui/icons/Photo'
import CloseIcon from '@material-ui/icons/Close'
import DeleteIcon from '@material-ui/icons/DeleteOutline'
import { httpClient } from '../dataProvider'
import { REST_URL } from '../consts'
import { baseUrl } from '../utils'

const PAGE_SIZE = 50

const useStyles = makeStyles((theme) => ({
  title: {
    fontWeight: 600,
    fontSize: '0.9rem',
    marginBottom: theme.spacing(1),
  },
  messages: {
    height: 380,
    overflowY: 'auto',
    display: 'flex',
    flexDirection: 'column',
    gap: theme.spacing(1),
    padding: theme.spacing(0.5, 0),
  },
  row: {
    display: 'flex',
    alignItems: 'flex-start',
    gap: theme.spacing(1),
    '&:hover $deleteBtn': {
      opacity: 1,
    },
  },
  body: { flex: 1, minWidth: 0 },
  header: {
    display: 'flex',
    alignItems: 'baseline',
    gap: theme.spacing(1),
  },
  userName: {
    fontWeight: 700,
    fontSize: '0.8rem',
  },
  time: {
    fontSize: '0.7rem',
    color: theme.palette.text.disabled,
  },
  text: {
    fontSize: '0.85rem',
    whiteSpace: 'pre-wrap',
    wordBreak: 'break-word',
  },
  image: {
    display: 'block',
    maxWidth: '100%',
    maxHeight: 240,
    borderRadius: theme.spacing(0.5),
    marginTop: theme.spacing(0.5),
    cursor: 'pointer',
  },
  deleteBtn: {
    opacity: 0,
    transition: 'opacity 0.2s',
    padding: theme.spacing(0.5),
  },
  empty: {
    fontSize: '0.8rem',
    color: theme.palette.text.disabled,
    margin: 'auto',
  },
  inputRow: {
    display: 'flex',
    alignItems: 'flex-end',
    gap: theme.spacing(0.5),
    marginTop: theme.spacing(1),
  },
  input: { flex: 1 },
  preview: {
    display: 'flex',
    alignItems: 'center',
    gap: theme.spacing(1),
    marginTop: theme.spacing(1),
  },
  previewImg: {
    maxHeight: 64,
    borderRadius: theme.spacing(0.5),
  },
  loadOlder: {
    alignSelf: 'center',
    flexShrink: 0,
  },
}))

const formatTime = (date) => {
  const d = new Date(date)
  const today = new Date()
  const sameDay = d.toDateString() === today.toDateString()
  return sameDay
    ? d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    : d.toLocaleDateString([], {
        month: 'short',
        day: 'numeric',
      }) +
        ` ${d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`
}

const chatImageUrl = (id) =>
  baseUrl(
    `${REST_URL}/community/chat/image/${id}?jwt=${localStorage.getItem(
      'token',
    )}`,
  )

const ChatBox = () => {
  const classes = useStyles()
  const translate = useTranslate()
  const notify = useNotify()
  const lastChatMessage = useSelector((state) => state.activity.lastChatMessage)

  // Messages held oldest-first for rendering
  const [messages, setMessages] = useState([])
  const [hasMore, setHasMore] = useState(false)
  const [text, setText] = useState('')
  const [pendingImage, setPendingImage] = useState(null)
  const [sending, setSending] = useState(false)

  const scrollRef = useRef(null)
  const stickToBottom = useRef(true)
  const fileInputRef = useRef(null)

  const userId = localStorage.getItem('userId')
  const isAdmin = localStorage.getItem('role') === 'admin'

  const previewUrl = useMemo(
    () => (pendingImage ? URL.createObjectURL(pendingImage) : null),
    [pendingImage],
  )
  useEffect(() => {
    return () => previewUrl && URL.revokeObjectURL(previewUrl)
  }, [previewUrl])

  const appendUnique = (prev, msg) =>
    prev.some((m) => m.id === msg.id) ? prev : [...prev, msg]

  useEffect(() => {
    httpClient(`${REST_URL}/community/chat?count=${PAGE_SIZE}`)
      .then((resp) => {
        const page = resp.json || []
        setMessages(page.slice().reverse())
        setHasMore(page.length === PAGE_SIZE)
      })
      .catch(() => {})
  }, [])

  const loadOlder = useCallback(() => {
    if (!messages.length) return
    const oldest = new Date(messages[0].createdAt).getTime()
    httpClient(`${REST_URL}/community/chat?count=${PAGE_SIZE}&before=${oldest}`)
      .then((resp) => {
        const page = resp.json || []
        stickToBottom.current = false
        setMessages((prev) => [...page.slice().reverse(), ...prev])
        setHasMore(page.length === PAGE_SIZE)
      })
      .catch(() => {})
  }, [messages])

  // Live updates via SSE
  useEffect(() => {
    if (!lastChatMessage) return
    if (lastChatMessage.deleted) {
      setMessages((prev) => prev.filter((m) => m.id !== lastChatMessage.id))
    } else {
      setMessages((prev) => appendUnique(prev, lastChatMessage))
    }
  }, [lastChatMessage])

  const handleScroll = useCallback((e) => {
    const el = e.target
    stickToBottom.current =
      el.scrollHeight - el.scrollTop - el.clientHeight < 60
  }, [])

  useEffect(() => {
    const el = scrollRef.current
    if (el && stickToBottom.current) {
      el.scrollTop = el.scrollHeight
    }
  }, [messages])

  const send = useCallback(async () => {
    const message = text.trim()
    if ((!message && !pendingImage) || sending) return
    setSending(true)
    try {
      const formData = new FormData()
      formData.append('message', message)
      if (pendingImage) {
        formData.append('image', pendingImage, pendingImage.name || 'pasted')
      }
      const resp = await httpClient(`${REST_URL}/community/chat`, {
        method: 'POST',
        headers: new Headers({}),
        body: formData,
      })
      stickToBottom.current = true
      if (resp.json) {
        setMessages((prev) => appendUnique(prev, resp.json))
      }
      setText('')
      setPendingImage(null)
    } catch (err) {
      notify('community.chat.sendError', 'warning')
    } finally {
      setSending(false)
    }
  }, [text, pendingImage, sending, notify])

  const handleKeyDown = useCallback(
    (e) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        send()
      }
    },
    [send],
  )

  const handlePaste = useCallback((e) => {
    const items = e.clipboardData && e.clipboardData.items
    if (!items) return
    for (const item of items) {
      if (item.kind === 'file' && item.type.startsWith('image/')) {
        const file = item.getAsFile()
        if (file) {
          e.preventDefault()
          setPendingImage(file)
          return
        }
      }
    }
  }, [])

  const handleFileChange = useCallback((e) => {
    const file = e.target.files[0]
    if (file) setPendingImage(file)
    e.target.value = ''
  }, [])

  const handleDelete = useCallback(
    async (id) => {
      try {
        await httpClient(`${REST_URL}/community/chat/${id}`, {
          method: 'DELETE',
        })
        setMessages((prev) => prev.filter((m) => m.id !== id))
      } catch (err) {
        notify('community.chat.deleteError', 'warning')
      }
    },
    [notify],
  )

  return (
    <Card>
      <CardContent>
        <Typography className={classes.title}>
          {translate('community.chat.title')}
        </Typography>
        <div
          className={classes.messages}
          ref={scrollRef}
          onScroll={handleScroll}
        >
          {hasMore && (
            <Button
              size="small"
              className={classes.loadOlder}
              onClick={loadOlder}
            >
              {translate('community.chat.loadOlder')}
            </Button>
          )}
          {messages.length === 0 ? (
            <Typography className={classes.empty}>
              {translate('community.chat.empty')}
            </Typography>
          ) : (
            messages.map((msg) => (
              <div key={msg.id} className={classes.row}>
                <div className={classes.body}>
                  <div className={classes.header}>
                    <span className={classes.userName}>{msg.userName}</span>
                    <span className={classes.time}>
                      {formatTime(msg.createdAt)}
                    </span>
                  </div>
                  {msg.message && (
                    <Typography component="div" className={classes.text}>
                      {msg.message}
                    </Typography>
                  )}
                  {msg.hasImage && (
                    <img
                      src={chatImageUrl(msg.id)}
                      alt=""
                      className={classes.image}
                      onClick={() =>
                        window.open(chatImageUrl(msg.id), '_blank')
                      }
                    />
                  )}
                </div>
                {(msg.userId === userId || isAdmin) && (
                  <Tooltip title={translate('community.chat.delete')}>
                    <IconButton
                      size="small"
                      className={classes.deleteBtn}
                      onClick={() => handleDelete(msg.id)}
                    >
                      <DeleteIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                )}
              </div>
            ))
          )}
        </div>
        {previewUrl && (
          <div className={classes.preview}>
            <img
              src={previewUrl}
              alt={translate('community.chat.attachedImage')}
              className={classes.previewImg}
            />
            <Tooltip title={translate('community.chat.removeImage')}>
              <IconButton size="small" onClick={() => setPendingImage(null)}>
                <CloseIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          </div>
        )}
        <div className={classes.inputRow}>
          <TextField
            className={classes.input}
            value={text}
            onChange={(e) => setText(e.target.value)}
            onKeyDown={handleKeyDown}
            onPaste={handlePaste}
            placeholder={translate('community.chat.placeholder')}
            multiline
            maxRows={4}
            variant="outlined"
            size="small"
          />
          <Tooltip title={translate('community.chat.attachImage')}>
            <IconButton
              size="small"
              onClick={() => fileInputRef.current?.click()}
            >
              <PhotoIcon />
            </IconButton>
          </Tooltip>
          <Tooltip title={translate('community.chat.send')}>
            <span>
              <IconButton
                size="small"
                color="primary"
                disabled={sending || (!text.trim() && !pendingImage)}
                onClick={send}
              >
                <SendIcon />
              </IconButton>
            </span>
          </Tooltip>
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*"
            style={{ display: 'none' }}
            onChange={handleFileChange}
          />
        </div>
      </CardContent>
    </Card>
  )
}

export default ChatBox
