import {
  EVENT_REFRESH_RESOURCE,
  EVENT_SCAN_STATUS,
  EVENT_SERVER_START,
  EVENT_NOW_PLAYING_COUNT,
  EVENT_NOW_PLAYING_COUNT_SYNC,
  EVENT_PLAY_EVENT,
  EVENT_RELEASE_ALERT,
  EVENT_STREAM_RECONNECTED,
} from '../actions'
import config from '../config'

const initialState = {
  scanStatus: {
    scanning: false,
    folderCount: 0,
    count: 0,
    error: '',
    elapsedTime: 0,
  },
  serverStart: { version: config.version },
  nowPlayingCount: 0,
  nowPlayingLastUpdate: 0,
  lastPlayEvent: null, // Last community play broadcast (scrobble)
  releaseAlerts: 0, // Bumped when new release alerts arrive via SSE
  streamReconnected: 0, // Timestamp of last reconnection
}

export const activityReducer = (previousState = initialState, payload) => {
  const { type, data } = payload

  switch (type) {
    case EVENT_SCAN_STATUS: {
      const elapsedTime = Number(data.elapsedTime) || 0
      return { ...previousState, scanStatus: { ...data, elapsedTime } }
    }
    case EVENT_SERVER_START:
      return {
        ...previousState,
        serverStart: {
          startTime: data.startTime && Date.parse(data.startTime),
          version: data.version,
        },
      }
    case EVENT_REFRESH_RESOURCE:
      return {
        ...previousState,
        refresh: {
          lastReceived: Date.now(),
          resources: data,
        },
      }
    case EVENT_NOW_PLAYING_COUNT:
      return {
        ...previousState,
        nowPlayingCount: data.count,
        nowPlayingLastUpdate: Date.now(),
      }
    case EVENT_NOW_PLAYING_COUNT_SYNC:
      return { ...previousState, nowPlayingCount: data.count }
    case EVENT_PLAY_EVENT:
      return {
        ...previousState,
        lastPlayEvent: { ...data, receivedAt: Date.now() },
      }
    case EVENT_RELEASE_ALERT:
      return { ...previousState, releaseAlerts: Date.now() }
    case EVENT_STREAM_RECONNECTED:
      return { ...previousState, streamReconnected: Date.now() }
    default:
      return previousState
  }
}
