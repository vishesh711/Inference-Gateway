'use client'

import { useState, useRef, useEffect } from 'react'
import axios from 'axios'

interface Message {
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp: Date
}

interface Stats {
  tokensGenerated: number
  generationTime: number
  queueTime: number
  requestsTotal: number
  cacheHits: number
}

export default function Home() {
  const [messages, setMessages] = useState<Message[]>([
    {
      role: 'system',
      content: 'Connected to Inference Gateway with token-aware scheduling.',
      timestamp: new Date()
    }
  ])
  const [input, setInput] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [stats, setStats] = useState<Stats>({
    tokensGenerated: 0,
    generationTime: 0,
    queueTime: 0,
    requestsTotal: 0,
    cacheHits: 0
  })
  const [streaming, setStreaming] = useState(true)
  const [temperature, setTemperature] = useState(0.7)
  const [maxTokens, setMaxTokens] = useState(150)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }

  useEffect(() => {
    scrollToBottom()
  }, [messages])

  // Fetch metrics
  useEffect(() => {
    const fetchMetrics = async () => {
      try {
        const response = await axios.get('/api/v1/health')
        // Parse metrics from health check or metrics endpoint
      } catch (error) {
        console.error('Failed to fetch metrics:', error)
      }
    }

    const interval = setInterval(fetchMetrics, 5000)
    return () => clearInterval(interval)
  }, [])

  const sendMessage = async () => {
    if (!input.trim() || isLoading) return

    const userMessage: Message = {
      role: 'user',
      content: input,
      timestamp: new Date()
    }

    setMessages(prev => [...prev, userMessage])
    setInput('')
    setIsLoading(true)

    const startTime = Date.now()

    try {
      if (streaming) {
        // Streaming response
        const response = await fetch('/api/v1/chat/completions', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            model: 'tinyllama',
            messages: [
              {
                role: 'user',
                content: input
              }
            ],
            max_tokens: maxTokens,
            temperature: temperature,
            stream: true
          }),
        })

        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`)
        }

        const reader = response.body?.getReader()
        const decoder = new TextDecoder()
        let accumulatedContent = ''

        const assistantMessage: Message = {
          role: 'assistant',
          content: '',
          timestamp: new Date()
        }
        setMessages(prev => [...prev, assistantMessage])

        while (true) {
          const { done, value } = await reader!.read()
          if (done) break

          const chunk = decoder.decode(value)
          const lines = chunk.split('\n')

          for (const line of lines) {
            if (line.startsWith('data: ')) {
              const data = line.slice(6)
              if (data === '[DONE]') continue

              try {
                const parsed = JSON.parse(data)
                const content = parsed.choices?.[0]?.delta?.content
                if (content) {
                  accumulatedContent += content
                  setMessages(prev => {
                    const newMessages = [...prev]
                    newMessages[newMessages.length - 1].content = accumulatedContent
                    return newMessages
                  })
                }
              } catch (e) {
                // Skip parsing errors
              }
            }
          }
        }

        const endTime = Date.now()
        setStats(prev => ({
          ...prev,
          tokensGenerated: prev.tokensGenerated + accumulatedContent.split(' ').length,
          generationTime: endTime - startTime,
          requestsTotal: prev.requestsTotal + 1
        }))

      } else {
        // Non-streaming response
        const response = await axios.post('/api/v1/completions', {
          model: 'tinyllama',
          prompt: input,
          max_tokens: maxTokens,
          temperature: temperature,
        })

        const assistantMessage: Message = {
          role: 'assistant',
          content: response.data.choices[0].text,
          timestamp: new Date()
        }

        setMessages(prev => [...prev, assistantMessage])

        const endTime = Date.now()
        setStats(prev => ({
          ...prev,
          tokensGenerated: prev.tokensGenerated + (response.data.usage?.completion_tokens || 0),
          generationTime: endTime - startTime,
          requestsTotal: prev.requestsTotal + 1
        }))
      }
    } catch (error: any) {
      console.error('Error:', error)
      const errorMessage: Message = {
        role: 'system',
        content: `Error: ${error.response?.data?.error || error.message || 'Failed to send message'}`,
        timestamp: new Date()
      }
      setMessages(prev => [...prev, errorMessage])
    } finally {
      setIsLoading(false)
    }
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      sendMessage()
    }
  }

  const clearChat = () => {
    setMessages([
      {
        role: 'system',
        content: 'Chat cleared. Connected to Inference Gateway.',
        timestamp: new Date()
      }
    ])
  }

  return (
    <div className="flex h-screen bg-gray-50 dark:bg-gray-900">
      {/* Sidebar */}
      <div className="w-80 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 p-6 flex flex-col">
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white mb-2">
            LLM Gateway
          </h1>
          <p className="text-sm text-gray-600 dark:text-gray-400">
            Token-aware scheduling
          </p>
        </div>

        {/* Stats */}
        <div className="space-y-4 mb-6">
          <div className="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-4">
            <div className="text-xs text-blue-600 dark:text-blue-400 mb-1">Total Requests</div>
            <div className="text-2xl font-bold text-blue-900 dark:text-blue-300">
              {stats.requestsTotal}
            </div>
          </div>

          <div className="bg-green-50 dark:bg-green-900/20 rounded-lg p-4">
            <div className="text-xs text-green-600 dark:text-green-400 mb-1">Tokens Generated</div>
            <div className="text-2xl font-bold text-green-900 dark:text-green-300">
              {stats.tokensGenerated}
            </div>
          </div>

          <div className="bg-purple-50 dark:bg-purple-900/20 rounded-lg p-4">
            <div className="text-xs text-purple-600 dark:text-purple-400 mb-1">Last Response</div>
            <div className="text-2xl font-bold text-purple-900 dark:text-purple-300">
              {stats.generationTime}ms
            </div>
          </div>
        </div>

        {/* Settings */}
        <div className="space-y-4">
          <div>
            <label className="flex items-center justify-between text-sm text-gray-700 dark:text-gray-300 mb-2">
              <span>Temperature: {temperature.toFixed(1)}</span>
            </label>
            <input
              type="range"
              min="0"
              max="2"
              step="0.1"
              value={temperature}
              onChange={(e) => setTemperature(parseFloat(e.target.value))}
              className="w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer dark:bg-gray-700"
            />
          </div>

          <div>
            <label className="flex items-center justify-between text-sm text-gray-700 dark:text-gray-300 mb-2">
              <span>Max Tokens: {maxTokens}</span>
            </label>
            <input
              type="range"
              min="10"
              max="500"
              step="10"
              value={maxTokens}
              onChange={(e) => setMaxTokens(parseInt(e.target.value))}
              className="w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer dark:bg-gray-700"
            />
          </div>

          <div className="flex items-center justify-between">
            <span className="text-sm text-gray-700 dark:text-gray-300">Streaming</span>
            <button
              onClick={() => setStreaming(!streaming)}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                streaming ? 'bg-blue-600' : 'bg-gray-300'
              }`}
            >
              <span
                className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                  streaming ? 'translate-x-6' : 'translate-x-1'
                }`}
              />
            </button>
          </div>
        </div>

        <button
          onClick={clearChat}
          className="mt-auto w-full bg-red-600 hover:bg-red-700 text-white rounded-lg py-2 px-4 transition-colors"
        >
          Clear Chat
        </button>
      </div>

      {/* Main Chat Area */}
      <div className="flex-1 flex flex-col">
        {/* Messages */}
        <div className="flex-1 overflow-y-auto p-6 space-y-4">
          {messages.map((message, index) => (
            <div
              key={index}
              className={`flex ${
                message.role === 'user' ? 'justify-end' : 'justify-start'
              }`}
            >
              <div
                className={`max-w-2xl rounded-lg px-4 py-3 ${
                  message.role === 'user'
                    ? 'bg-blue-600 text-white'
                    : message.role === 'system'
                    ? 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 text-sm'
                    : 'bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border border-gray-200 dark:border-gray-700'
                }`}
              >
                <div className="whitespace-pre-wrap">{message.content}</div>
                <div className={`text-xs mt-2 ${
                  message.role === 'user' ? 'text-blue-200' : 'text-gray-500 dark:text-gray-400'
                }`}>
                  {message.timestamp.toLocaleTimeString()}
                </div>
              </div>
            </div>
          ))}
          {isLoading && (
            <div className="flex justify-start">
              <div className="bg-white dark:bg-gray-800 rounded-lg px-4 py-3 border border-gray-200 dark:border-gray-700">
                <div className="flex space-x-2">
                  <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce"></div>
                  <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '0.1s' }}></div>
                  <div className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '0.2s' }}></div>
                </div>
              </div>
            </div>
          )}
          <div ref={messagesEndRef} />
        </div>

        {/* Input Area */}
        <div className="border-t border-gray-200 dark:border-gray-700 p-4 bg-white dark:bg-gray-800">
          <div className="flex space-x-4">
            <textarea
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyPress={handleKeyPress}
              placeholder="Type your message... (Shift+Enter for new line)"
              className="flex-1 resize-none rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 px-4 py-3 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
              rows={3}
              disabled={isLoading}
            />
            <button
              onClick={sendMessage}
              disabled={isLoading || !input.trim()}
              className="px-6 py-3 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed text-white rounded-lg font-semibold transition-colors"
            >
              {isLoading ? 'Sending...' : 'Send'}
            </button>
          </div>
          <div className="mt-2 text-xs text-gray-500 dark:text-gray-400 text-center">
            Gateway running with bounded queue + token-aware scheduling
          </div>
        </div>
      </div>
    </div>
  )
}
