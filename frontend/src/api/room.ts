import { apiFetch } from './client'

export interface CreateRoomResponse {
  roomCode: string
  matchId: string
}

export interface JoinRoomResponse {
  matchId: string
}

export interface RoomStatusResponse {
  matchId: string
  playerCount: number
  status: string
}

export async function createRoom(userId: string, username: string): Promise<CreateRoomResponse> {
  return apiFetch<CreateRoomResponse>('/rooms', {
    method: 'POST',
    body: JSON.stringify({ userId, username }),
  })
}

export async function joinRoom(code: string, userId: string, username: string): Promise<JoinRoomResponse> {
  return apiFetch<JoinRoomResponse>(`/rooms/${code}/join`, {
    method: 'POST',
    body: JSON.stringify({ userId, username }),
  })
}

export async function getRoomStatus(code: string): Promise<RoomStatusResponse> {
  return apiFetch<RoomStatusResponse>(`/rooms/${code}`)
}
