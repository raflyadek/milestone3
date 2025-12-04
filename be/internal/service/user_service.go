package service

import (
	"errors"
	"log"

	"golang.org/x/crypto/bcrypt"

	"milestone3/be/internal/dto"
	"milestone3/be/internal/entity"
	"milestone3/be/internal/utils"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *entity.Users) error
	GetByEmail(email string) (user entity.Users, err error)
	GetById(id int) (user entity.Users, err error)
}

type UserServ struct {
	userRepo UserRepository
}

func NewUserService(ur UserRepository) *UserServ {
	return &UserServ{userRepo: ur}
}

func (us *UserServ) CreateUser(req dto.UserRequest) (res dto.UserResponse, err error) {
	passHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("error encrypt password")
		return dto.UserResponse{}, err
	}

	req.Password = string(passHash)

	user := entity.Users{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	}

	if err := us.userRepo.Create(&user); err != nil {
		return dto.UserResponse{}, err
	}

	//get user id to show in the response
	userInfo, err := us.GetUserById(user.Id)
	if err != nil {
		log.Println("failed get user by id")
		return dto.UserResponse{}, err
	}

	return userInfo, nil
}

func (us *UserServ) GetUserById(id int) (res dto.UserResponse, err error) {
	user, err := us.userRepo.GetById(id)
	if err != nil {
		log.Println("failed get user by id")
		return dto.UserResponse{}, err
	}

	userInfo := dto.UserResponse{
		Id:    user.Id,
		Name:  user.Name,
		Email: user.Email,
		Role:  user.Role,
	}

	return userInfo, nil
}

func (us *UserServ) GetUserByEmail(email, password string) (accessToken string, err error) {
	log.Printf("[DEBUG] Login attempt for email: %s", email)
	
	user, err := us.userRepo.GetByEmail(email)
	if err != nil {
		log.Printf("[ERROR] Failed get user by email: %v", err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrInvalidCredential
		}
		return "", err
	}

	log.Printf("[DEBUG] User found: ID=%d, Email=%s, Role=%s", user.Id, user.Email, user.Role)
	log.Printf("[DEBUG] Stored hash length: %d, Input password length: %d", len(user.Password), len(password))

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		log.Printf("[ERROR] Password comparison failed: %v", err)
		return "", ErrInvalidCredential
	}

	log.Printf("[DEBUG] Password verified, generating token...")
	token, err := utils.GenerateJwtToken(email, user.Role, user.Id)
	if err != nil {
		log.Printf("[ERROR] Failed generate jwt token: %v", err)
		return "", err
	}

	log.Printf("[DEBUG] Login successful for user ID: %d", user.Id)
	return token, nil
}
