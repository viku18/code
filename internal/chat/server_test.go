package chat

import (
	"context"
	"testing"

	"github.com/acai-travel/tech-challenge/internal/chat/model"
	. "github.com/acai-travel/tech-challenge/internal/chat/testing"
	"github.com/acai-travel/tech-challenge/internal/pb"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/mock"
	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestServer_StartConversation(t *testing.T) {
	ctx := context.Background()

	t.Run("creates new conversation with populated title and triggers assistant response", WithFixture(func(t *testing.T, f *Fixture) {
		// Create a mock assistant service
		mockAssistant := new(MockAssistantService)
		srv := NewServer(model.New(ConnectMongo()), mockAssistant)

		// Set up expectations for the assistant response
		mockAssistant.On("Respond", mock.Anything, mock.Anything).
			Return(&pb.Message{
				Id:      "assistant-msg-1",
				Content: "Hello! How can I help you today?",
				Role:    pb.Role_ROLE_ASSISTANT,
			}, nil)

		// Call StartConversation
		req := &pb.StartConversationRequest{
			Message: &pb.Message{
				Content: "Hello, I need help with my travel plans",
				Role:    pb.Role_ROLE_USER,
			},
		}

		resp, err := srv.StartConversation(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify the response contains a conversation ID
		if resp.GetConversationId() == "" {
			t.Error("expected non-empty conversation ID")
		}

		// Verify the conversation was created in the database
		conversation, err := srv.model.GetConversation(ctx, resp.GetConversationId())
		if err != nil {
			t.Fatalf("failed to get conversation from database: %v", err)
		}

		// Verify the conversation has a populated title
		if conversation.Title == "" {
			t.Error("expected conversation to have a populated title")
		}

		// Verify the title is derived from the user message
		expectedTitle := "Hello, I need help with my travel plans"
		if conversation.Title != expectedTitle {
			t.Errorf("expected title %q, got %q", expectedTitle, conversation.Title)
		}

		// Verify the conversation contains the user message
		if len(conversation.Messages) < 1 {
			t.Error("expected conversation to contain at least one message")
		}

		userMessage := conversation.Messages[0]
		if userMessage.Content != req.Message.Content {
			t.Errorf("expected user message content %q, got %q", req.Message.Content, userMessage.Content)
		}
		if userMessage.Role != pb.Role_ROLE_USER {
			t.Errorf("expected user message role ROLE_USER, got %v", userMessage.Role)
		}

		// Verify the assistant response was triggered and added to the conversation
		if len(conversation.Messages) < 2 {
			t.Error("expected conversation to contain assistant response")
		}

		assistantMessage := conversation.Messages[1]
		if assistantMessage.Role != pb.Role_ROLE_ASSISTANT {
			t.Errorf("expected assistant message role ROLE_ASSISTANT, got %v", assistantMessage.Role)
		}
		if assistantMessage.Content == "" {
			t.Error("expected assistant message to have content")
		}

		// Verify the mock assistant was called
		mockAssistant.AssertCalled(t, "Respond", mock.Anything, mock.Anything)
	}))

	t.Run("handles empty message", WithFixture(func(t *testing.T, f *Fixture) {
		mockAssistant := new(MockAssistantService)
		srv := NewServer(model.New(ConnectMongo()), mockAssistant)

		req := &pb.StartConversationRequest{
			Message: &pb.Message{
				Content: "",
				Role:    pb.Role_ROLE_USER,
			},
		}

		_, err := srv.StartConversation(ctx, req)
		if err == nil {
			t.Fatal("expected error for empty message, got nil")
		}

		if te, ok := err.(twirp.Error); !ok || te.Code() != twirp.InvalidArgument {
			t.Fatalf("expected twirp.InvalidArgument error, got %v", err)
		}

		// Verify assistant was not called for invalid input
		mockAssistant.AssertNotCalled(t, "Respond")
	}))

	t.Run("handles assistant service error", WithFixture(func(t *testing.T, f *Fixture) {
		mockAssistant := new(MockAssistantService)
		srv := NewServer(model.New(ConnectMongo()), mockAssistant)

		// Mock assistant to return an error
		mockAssistant.On("Respond", mock.Anything, mock.Anything).
			Return(nil, twirp.InternalError("assistant service unavailable"))

		req := &pb.StartConversationRequest{
			Message: &pb.Message{
				Content: "Test message",
				Role:    pb.Role_ROLE_USER,
			},
		}

		_, err := srv.StartConversation(ctx, req)
		if err == nil {
			t.Fatal("expected error from assistant service, got nil")
		}

		// Verify the conversation was still created despite assistant error
		mockAssistant.AssertCalled(t, "Respond", mock.Anything, mock.Anything)
	}))
}

// MockAssistantService is a mock implementation of AssistantService for testing
type MockAssistantService struct {
	mock.Mock
}

func (m *MockAssistantService) Respond(ctx context.Context, conversation *model.Conversation) (*pb.Message, error) {
	args := m.Called(ctx, conversation)
	if msg := args.Get(0); msg != nil {
		return msg.(*pb.Message), args.Error(1)
	}
	return nil, args.Error(1)
}