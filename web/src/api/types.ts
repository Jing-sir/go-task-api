export interface ApiEnvelope<T> {
  code: number
  message: string
  data: T

}

export interface TaskEvent {
  data: Task | number
  type: 'task.created' | 'task.updated' | 'task.deleted'
}

export interface User {
  id: number
  email: string
  avatar: string
  created_at: string
  updated_at: string
}

export interface AuthPayload {
  email: string
  password: string
}

export interface LoginResponse {
  token: string
  user: User
}

export interface Task {
  id: number
  user_id: number
  title: string
  description: string
  status: boolean
  created_at: string
  updated_at: string
}

export interface CreateTaskPayload {
  title: string
  description: string
  status: boolean
}

export interface TaskListResponse {
  list: Task[]
  total: number
  page_no: number
  page_size: number
}

